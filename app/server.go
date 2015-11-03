/*
The MIT License (MIT)

Copyright (c) 2013-2015 SRS(simple-rtmp-server)

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package app

import (
	"fmt"
	"github.com/simple-rtmp-server/go-srs/core"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	sigs   chan os.Signal
	quit   chan chan error
	wg     sync.WaitGroup
	logger *simpleLogger
}

func NewServer() *Server {
	svr := &Server{
		sigs:   make(chan os.Signal, 1),
		quit:   make(chan chan error, 1),
		logger: &simpleLogger{},
	}

	GsConfig.Subscribe(svr)

	return svr
}

func (s *Server) Close() {
	GsConfig.Unsubscribe(s)
	// TODO: FIXME: do cleanup.
}

func (s *Server) ParseConfig(conf string) (err error) {
	core.GsTrace.Println("start to parse config file", conf)
	if err = GsConfig.Loads(conf); err != nil {
		return
	}

	return
}

func (s *Server) PrepareLogger() (err error) {
	if err = s.applyLogger(GsConfig); err != nil {
		return
	}

	return
}

func (s *Server) Initialize() (err error) {
	// install signals.
	// TODO: FIXME: when process the current signal, others may drop.
	signal.Notify(s.sigs)

	// reload goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		configReloadWorker(s.quit)
		core.GsTrace.Println("reload worker terminated.")
	}()

	c := GsConfig
	l := fmt.Sprintf("%v(%v/%v)", c.Log.Tank, c.Log.Level, c.Log.File)
	if !c.LogToFile() {
		l = fmt.Sprintf("%v(%v)", c.Log.Tank, c.Log.Level)
	}
	core.GsTrace.Println(fmt.Sprintf("init server ok, conf=%v, log=%v, workers=%v, gc=%v", c.conf, l, c.Workers, c.Go.GcInterval))

	return
}

func (s *Server) Run() (err error) {
	s.applyMultipleProcesses(GsConfig.Workers)

	for {
		select {
		case signal := <-s.sigs:
			core.GsTrace.Println("got signal", signal)
			switch signal {
			case os.Interrupt:
				// SIGINT
				fallthrough
			case syscall.SIGTERM:
				// SIGTERM
				q := make(chan error, 1)
				s.quit <- q
			}
		case q := <-s.quit:
			s.quit <- q
			s.wg.Wait()
			core.GsWarn.Println("server quit")
			return
		case <-time.After(time.Second * time.Duration(GsConfig.Go.GcInterval)):
			runtime.GC()
			core.GsError.Println("go runtime gc every", GsConfig.Go.GcInterval, "seconds")
		}
	}

	return
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
