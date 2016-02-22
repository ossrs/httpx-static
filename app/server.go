// The MIT License (MIT)
//
// Copyright (c) 2013-2016 Oryx(ossrs)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package app

import (
	"fmt"
	"github.com/ossrs/go-oryx/agent"
	"github.com/ossrs/go-oryx/core"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"
)

// the state of server, state graph:
//      Init => Normal(Ready => Running)
//      Init/Normal => Closed
type ServerState int

const (
	StateInit ServerState = 1 << iota
	StateReady
	StateRunning
	StateClosed
)

type Server struct {
	ctx core.Context
	// signal handler.
	sigs chan os.Signal
	// whether closed.
	closed  ServerState
	closing chan bool
	// for system internal to notify quit.
	quit chan bool
	wg   sync.WaitGroup
	// core components.
	htbt   *Heartbeat
	logger *simpleLogger
	rtmp   core.OpenCloser
	// the locker for state, for instance, the closed.
	lock sync.Mutex
}

func NewServer(ctx core.Context) *Server {
	v := &Server{
		ctx:     ctx,
		sigs:    make(chan os.Signal, 1),
		closed:  StateInit,
		closing: make(chan bool, 1),
		quit:    make(chan bool, 1),
		htbt:    NewHeartbeat(ctx),
		logger:  &simpleLogger{ctx: ctx},
	}
	v.rtmp = agent.NewRtmp(ctx, v)

	core.Conf.Subscribe(v)

	return v
}

// notify server to stop and wait for cleanup.
func (v *Server) Close() {
	ctx := v.ctx

	// wait for stopped.
	v.lock.Lock()
	defer v.lock.Unlock()

	// only create?
	if v.closed == StateInit {
		return
	}

	// closed?
	if v.closed == StateClosed {
		core.Warn.Println(ctx, "server already closed.")
		return
	}

	// notify to close.
	if v.closed == StateRunning {
		core.Info.Println(ctx, "notify server to stop.")
		select {
		case v.quit <- true:
		default:
		}
	}

	// wait for closed.
	if v.closed == StateRunning {
		<-v.closing
	}

	// do cleanup when stopped.
	core.Conf.Unsubscribe(v)

	// close the rtmp agent.
	if err := v.rtmp.Close(); err != nil {
		core.Warn.Println(ctx, "close rtmp agent failed. err is", err)
	}

	// close the agent manager.
	agent.Manager.Close()

	// when cpu profile is enabled, close it.
	if core.Conf.Go.CpuProfile != "" {
		pprof.StopCPUProfile()
		core.Trace.Println(ctx, "cpu profile ok, file is", core.Conf.Go.CpuProfile)
	}

	// when memory profile enabled, write heap info.
	if core.Conf.Go.MemProfile != "" {
		if f, err := os.Create(core.Conf.Go.MemProfile); err != nil {
			core.Warn.Println(ctx, "ignore open memory profile failed. err is", err)
		} else {
			defer f.Close()
			if err = pprof.Lookup("heap").WriteTo(f, 0); err != nil {
				core.Warn.Println(ctx, "write memory profile failed. err is", err)
			}
		}
		core.Trace.Println(ctx, "mem profile ok, file is", core.Conf.Go.MemProfile)
	}

	// ok, closed.
	v.closed = StateClosed
	core.Trace.Println(ctx, "server closed")
}

func (v *Server) ParseConfig(conf string) (err error) {
	ctx := v.ctx

	v.lock.Lock()
	defer v.lock.Unlock()

	if v.closed != StateInit {
		panic("server invalid state.")
	}

	core.Trace.Println(ctx, "start to parse config file", conf)
	if err = core.Conf.Loads(conf); err != nil {
		return
	}

	return
}

func (v *Server) PrepareLogger() (err error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.closed != StateInit {
		panic("server invalid state.")
	}

	if err = v.applyLogger(core.Conf); err != nil {
		return
	}

	return
}

func (v *Server) initializeRuntime() (err error) {
	ctx := v.ctx

	// install signals.
	// TODO: FIXME: when process the current signal, others may drop.
	signal.Notify(v.sigs)

	// apply the cpu profile.
	if err = v.applyCpuProfile(core.Conf); err != nil {
		return
	}

	// apply the gc percent.
	if err = v.applyGcPercent(core.Conf); err != nil {
		return
	}

	// show gc trace.
	go func() {
		stat := &debug.GCStats{}

		for {
			if core.Conf.Go.GcTrace > 0 {
				pgc := stat.NumGC
				debug.ReadGCStats(stat)
				if len(stat.Pause) > 3 {
					stat.Pause = append([]time.Duration{}, stat.Pause[:3]...)
				}
				if pgc < stat.NumGC {
					core.Trace.Println(ctx, "gc", stat.NumGC, stat.PauseTotal, stat.Pause, stat.PauseQuantiles)
				}
				time.Sleep(time.Duration(core.Conf.Go.GcTrace) * time.Second)
			} else {
				time.Sleep(3 * time.Second)
			}
		}
	}()

	return
}

func (v *Server) Initialize() (err error) {
	ctx := v.ctx

	v.lock.Lock()
	defer v.lock.Unlock()

	if v.closed != StateInit {
		panic("server invalid state.")
	}

	// about the runtime.
	if err = v.initializeRuntime(); err != nil {
		return
	}

	// use worker container to fork.
	var wc core.WorkerContainer = v

	// heartbeat with http api.
	if err = v.htbt.Initialize(wc); err != nil {
		return
	}

	// reload goroutine
	wc.GFork("reload", core.Conf.ReloadCycle)
	// heartbeat goroutine
	wc.GFork("htbt(discovery)", v.htbt.discoveryCycle)
	wc.GFork("htbt(main)", v.htbt.beatCycle)
	// open rtmp agent.
	if err = v.rtmp.Open(); err != nil {
		core.Error.Println(ctx, "open rtmp agent failed. err is", err)
		return
	}

	c := core.Conf
	l := fmt.Sprintf("%v(%v/%v)", c.Log.Tank, c.Log.Level, c.Log.File)
	if !c.LogToFile() {
		l = fmt.Sprintf("%v(%v)", c.Log.Tank, c.Log.Level)
	}
	core.Trace.Println(ctx, fmt.Sprintf("init server ok, conf=%v, log=%v, workers=%v/%v, gc=%v/%v%%, daemon=%v",
		c.Conf(), l, c.Workers, runtime.NumCPU(), c.Go.GcInterval, c.Go.GcPercent, c.Daemon))

	// set to ready, requires cleanup.
	v.closed = StateReady

	return
}

func (v *Server) onSignal(signal os.Signal) {
	ctx := v.ctx
	wc := v

	core.Trace.Println(ctx, "got signal", signal)
	switch signal {
	case SIGUSR1, SIGUSR2:
		panic("panic by SIGUSR1/2")
	case os.Interrupt, syscall.SIGTERM:
		// SIGINT, SIGTERM
		wc.Quit()
	}
}

func (v *Server) Run() (err error) {
	ctx := v.ctx

	func() {
		v.lock.Lock()
		defer v.lock.Unlock()

		if v.closed != StateReady {
			panic("server invalid state.")
		}
		v.closed = StateRunning
	}()

	// when terminated, notify the chan.
	defer func() {
		select {
		case v.closing <- true:
		default:
		}
	}()

	core.Info.Println(ctx, "server cycle running")

	// run server, apply settings.
	v.applyMultipleProcesses(core.Conf.Workers)

	var wc core.WorkerContainer = v
	for {
		var gcc <-chan time.Time = nil
		if core.Conf.Go.GcInterval > 0 {
			gcc = time.After(time.Second * time.Duration(core.Conf.Go.GcInterval))
		}

		select {
		case signal := <-v.sigs:
			v.onSignal(signal)
		case <-wc.QC():
			wc.Quit()

			// for the following quit will block all signal process,
			// we start new goroutine to process the panic signal only.
			go func() {
				for s := range v.sigs {
					v.onSignal(s)
				}
			}()

			// wait for all goroutines quit.
			v.wg.Wait()
			core.Warn.Println(ctx, "server cycle ok")
			return
		case <-gcc:
			runtime.GC()
			core.Info.Println(ctx, "go runtime gc every", core.Conf.Go.GcInterval, "seconds")
		}
	}

	return
}

// interface WorkContainer
func (v *Server) QC() <-chan bool {
	return v.quit
}

func (v *Server) Quit() error {
	select {
	case v.quit <- true:
	default:
	}

	return core.QuitError
}

func (v *Server) GFork(name string, f func(core.WorkerContainer)) {
	ctx := v.ctx

	v.wg.Add(1)
	go func() {
		defer v.wg.Done()

		defer func() {
			if r := recover(); r != nil {
				if !core.IsNormalQuit(r) {
					core.Warn.Println(ctx, "rtmp ignore", r)
				}

				core.Error.Println(ctx, string(debug.Stack()))

				v.Quit()
			}
		}()

		f(v)

		if name != "" {
			core.Trace.Println(ctx, name, "worker terminated.")
		}
	}()
}

func (v *Server) applyMultipleProcesses(workers int) {
	ctx := v.ctx

	if workers < 0 {
		panic("should not be negative workers")
	}

	if workers == 0 {
		workers = runtime.NumCPU()
	}
	pv := runtime.GOMAXPROCS(workers)

	core.Trace.Println(ctx, "apply workers", workers, "and previous is", pv)
}

func (v *Server) applyLogger(c *core.Config) (err error) {
	ctx := v.ctx

	if err = v.logger.close(c); err != nil {
		return
	}
	core.Info.Println(ctx, "close logger ok")

	if err = v.logger.open(c); err != nil {
		return
	}
	core.Info.Println(ctx, "open logger ok")

	return
}

func (v *Server) applyCpuProfile(c *core.Config) (err error) {
	ctx := v.ctx

	pprof.StopCPUProfile()

	if c.Go.CpuProfile == "" {
		return
	}

	var f *os.File
	if f, err = os.Create(c.Go.CpuProfile); err != nil {
		core.Error.Println(ctx, "open cpu profile file failed. err is", err)
		return
	}
	if err = pprof.StartCPUProfile(f); err != nil {
		core.Error.Println(ctx, "start cpu profile failed. err is", err)
		return
	}
	return
}

func (v *Server) applyGcPercent(c *core.Config) (err error) {
	ctx := v.ctx

	if c.Go.GcPercent == 0 {
		debug.SetGCPercent(100)
		return
	}

	pv := debug.SetGCPercent(c.Go.GcPercent)
	core.Trace.Println(ctx, "set gc percent from", pv, "to", c.Go.GcPercent)
	return
}

// interface ReloadHandler
func (v *Server) OnReloadGlobal(scope int, cc, pc *core.Config) (err error) {
	if scope == core.ReloadWorkers {
		v.applyMultipleProcesses(cc.Workers)
	} else if scope == core.ReloadLog {
		v.applyLogger(cc)
	} else if scope == core.ReloadCpuProfile {
		v.applyCpuProfile(cc)
	} else if scope == core.ReloadGcPercent {
		v.applyGcPercent(cc)
	}

	return
}

func (v *Server) OnReloadVhost(vhost string, scope int, cc, pc *core.Config) (err error) {
	return
}
