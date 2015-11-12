// The MIT License (MIT)
//
// Copyright (c) 2013-2015 SRS(ossrs)
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
	"github.com/ossrs/go-srs/core"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// the container for all worker,
// which provides the quit and cleanup methods.
type WorkerContainer interface {
	// get the quit channel,
	// worker can fetch the quit signal.
	// please use Quit to notify the container to quit.
	QC() <-chan bool
	// notify the container to quit.
	// for example, when goroutine fatal error,
	// which can't be recover, notify server to cleanup and quit.
	// @remark when got quit signal, the goroutine must notify the
	//      container to Quit(), for which others goroutines wait.
	Quit()
	// fork a new goroutine with work container.
	// the param f can be a global func or object method.
	// the param name is the goroutine name.
	GFork(name string, f func(WorkerContainer))
}

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
	// the locker for state, for instance, the closed.
	lock sync.Mutex
}

func NewServer() *Server {
	svr := &Server{
		sigs:    make(chan os.Signal, 1),
		closed:  StateInit,
		closing: make(chan bool, 1),
		quit:    make(chan bool, 1),
		htbt:    NewHeartbeat(),
		logger:  &simpleLogger{},
	}

	GsConfig.Subscribe(svr)

	return svr
}

// notify server to stop and wait for cleanup.
func (s *Server) Close() {
	// wait for stopped.
	s.lock.Lock()
	defer s.lock.Unlock()

	// closed?
	if s.closed == StateClosed {
		core.GsInfo.Println("server already closed.")
		return
	}

	// notify to close.
	if s.closed == StateRunning {
		core.GsInfo.Println("notify server to stop.")
		select {
		case s.quit <- true:
		default:
		}
	}

	// wait for closed.
	if s.closed == StateRunning {
		<-s.closing
	}

	// do cleanup when stopped.
	GsConfig.Unsubscribe(s)

	// ok, closed.
	s.closed = StateClosed
	core.GsInfo.Println("server closed")
}

func (s *Server) ParseConfig(conf string) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed != StateInit {
		panic("server invalid state.")
	}
	s.closed = StateReady

	core.GsTrace.Println("start to parse config file", conf)
	if err = GsConfig.Loads(conf); err != nil {
		return
	}

	return
}

func (s *Server) PrepareLogger() (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed != StateReady {
		panic("server invalid state.")
	}

	if err = s.applyLogger(GsConfig); err != nil {
		return
	}

	return
}

func (s *Server) Initialize() (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed != StateReady {
		panic("server invalid state.")
	}

	// install signals.
	// TODO: FIXME: when process the current signal, others may drop.
	signal.Notify(s.sigs)

	// reload goroutine
	s.GFork("reload", GsConfig.reloadCycle)
	// heartbeat goroutine
	s.GFork("htbt(discovery)", s.htbt.discoveryCycle)
	s.GFork("htbt(main)", s.htbt.beatCycle)

	c := GsConfig
	l := fmt.Sprintf("%v(%v/%v)", c.Log.Tank, c.Log.Level, c.Log.File)
	if !c.LogToFile() {
		l = fmt.Sprintf("%v(%v)", c.Log.Tank, c.Log.Level)
	}
	core.GsTrace.Println(fmt.Sprintf("init server ok, conf=%v, log=%v, workers=%v, gc=%v, daemon=%v",
		c.conf, l, c.Workers, c.Go.GcInterval, c.Daemon))

	return
}

func (s *Server) Run() (err error) {
	func() {
		s.lock.Lock()
		defer s.lock.Unlock()

		if s.closed != StateReady {
			panic("server invalid state.")
		}
		s.closed = StateRunning
	}()

	// when terminated, notify the chan.
	defer func() {
		select {
		case s.closing <- true:
		default:
		}
	}()

	core.GsInfo.Println("server running")

	// run server, apply settings.
	s.applyMultipleProcesses(GsConfig.Workers)

	for {
		select {
		case signal := <-s.sigs:
			core.GsTrace.Println("got signal", signal)
			switch signal {
			case os.Interrupt, syscall.SIGTERM:
				// SIGINT, SIGTERM
				s.Quit()
			}
		case <-s.QC():
			s.Quit()
			s.wg.Wait()
			core.GsWarn.Println("server quit")
			return
		case <-time.After(time.Second * time.Duration(GsConfig.Go.GcInterval)):
			runtime.GC()
			core.GsInfo.Println("go runtime gc every", GsConfig.Go.GcInterval, "seconds")
		}
	}

	return
}

// interface WorkContainer
func (s *Server) QC() <-chan bool {
	return s.quit
}

func (s *Server) Quit() {
	select {
	case s.quit <- true:
	default:
	}
}

func (s *Server) GFork(name string, f func(WorkerContainer)) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		defer func() {
			if r := recover(); r != nil {
				core.GsError.Println(name, "worker panic:", r)
				s.Quit()
			}
		}()

		f(s)
		core.GsTrace.Println(name, "worker terminated.")
	}()
}

// interface ReloadHandler
func (s *Server) OnReloadGlobal(scope int, cc, pc *Config) (err error) {
	if scope == ReloadWorkers {
		s.applyMultipleProcesses(cc.Workers)
	} else if scope == ReloadLog {
		s.applyLogger(cc)
	}

	return
}

func (s *Server) applyMultipleProcesses(workers int) {
	pv := runtime.GOMAXPROCS(workers)

	if pv != workers {
		core.GsTrace.Println("apply workers", workers, "and previous is", pv)
	} else {
		core.GsInfo.Println("apply workers", workers, "and previous is", pv)
	}
}

func (s *Server) applyLogger(c *Config) (err error) {
	if err = s.logger.close(c); err != nil {
		return
	}
	core.GsInfo.Println("close logger ok")

	if err = s.logger.open(c); err != nil {
		return
	}
	core.GsInfo.Println("open logger ok")

	return
}
