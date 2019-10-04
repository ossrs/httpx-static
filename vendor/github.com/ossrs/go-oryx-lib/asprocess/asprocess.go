// The MIT License (MIT)
//
// Copyright (c) 2013-2017 Oryx(ossrs)
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

// The oryx asprocess package provides associated process, which fork by parent
// process and die when parent die, for example, BMS server use asprocess to
// transcode audio, resolve DNS, bridge protocol(redis, kafka e.g.), and so on.
package asprocess

import (
	ol "github.com/ossrs/go-oryx-lib/logger"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// The recomment interval to check the parent pid.
const CheckParentInterval = time.Second * 1
const Interval = CheckParentInterval

// The cleanup function.
type Cleanup func()

// Watch the parent process, listen singals, quit when parent quit or signal quit.
// @remark optional ctx the logger context. nil to ignore.
// @reamrk check interval, user can use const CheckParentInterval
// @remark optional callback cleanup callback function. nil to ignore.
func Watch(ctx ol.Context, interval time.Duration, callback Cleanup) {
	v := &aspContext{
		ctx:      ctx,
		interval: interval,
		callback: callback,
	}

	v.InstallSignals()

	v.WatchParent()
}

// Watch the parent process only, write to quit when parent changed.
// This is used for asprocess which need to control the quit workflow and signals.
// @remark quit should be make(chan bool, 1) to write quit signal, drop when write failed.
// @reamrk check interval, user can use const CheckParentInterval
// @remark user should never close the quit, or watcher will panic when write to closed chan.
func WatchNoExit(ctx ol.Context, interval time.Duration, quit chan<- bool) {
	v := aspContextNoExit{
		ctx:      ctx,
		interval: interval,
		quit:     quit,
	}

	v.WatchParent()
}

type aspContext struct {
	ctx      ol.Context
	interval time.Duration
	callback Cleanup
}

func (v *aspContext) InstallSignals() {
	// install signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for s := range sigs {
			ol.T(v.ctx, "go signal", s)

			if v.callback != nil {
				v.callback()
			}

			os.Exit(0)
		}
	}()
	ol.T(v.ctx, "signal watched by asprocess")
}

func (v *aspContext) WatchParent() {
	ppid := os.Getppid()

	// If parent is 1 when start, ignore.
	if ppid == 1 {
		ol.T(v.ctx, "ignore parent event for ppid is 1")
		return
	}

	go func() {
		for {
			if pid := os.Getppid(); pid == 1 || pid != ppid {
				ol.E(v.ctx, "quit for parent problem, ppid is", pid)

				if v.callback != nil {
					v.callback()
				}

				os.Exit(0)
			}
			//ol.T(v.ctx, "parent pid", ppid, "ok")

			time.Sleep(v.interval)
		}
	}()
	ol.T(v.ctx, "parent process watching, ppid is", ppid)
}

type aspContextNoExit struct {
	ctx      ol.Context
	interval time.Duration
	quit     chan<- bool
}

func (v *aspContextNoExit) WatchParent() {
	ppid := os.Getppid()

	// If parent is 1 when start, ignore.
	if ppid == 1 {
		ol.T(v.ctx, "ignore parent event for ppid is 1")
		return
	}

	go func() {
		for {
			if pid := os.Getppid(); pid == 1 || pid != ppid {
				ol.E(v.ctx, "quit for parent problem, ppid is", pid)
				select {
				case v.quit <- true:
				default:
				}
				break
			}
			//ol.T(v.ctx, "parent pid", ppid, "ok")

			time.Sleep(v.interval)
		}
	}()
	ol.T(v.ctx, "parent process watching, ppid is", ppid)
}
