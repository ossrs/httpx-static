/*
The MIT License (MIT)

Copyright (c) 2016 winlin

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

/*
 This is the sync objects for multiple goroutine to work together.
*/
package kernel

import (
	ol "github.com/ossrs/go-oryx-lib/logger"
	"os"
	"os/signal"
	"sync"
)

// a group of worker, each is a goroutine.
type WorkerGroup struct {
	closing chan bool
	wait    *sync.WaitGroup
	closed  bool
}

func NewWorkerGroup() *WorkerGroup {
	return &WorkerGroup{
		closing: make(chan bool, 1),
		wait:    &sync.WaitGroup{},
	}
}

// when got singal from this chan, quit.
func (v *WorkerGroup) QuitForChan(closing chan bool) {
	go func() {
		for _ = range closing {
			v.Close()
		}
	}()
}

// quit when got these signals.
func (v *WorkerGroup) QuitForSignals(ctx ol.Context, signals ...os.Signal) {
	go func() {
		ss := make(chan os.Signal)
		signal.Notify(ss, signals...)
		for s := range ss {
			ol.W(ctx, "quit for signal", s)
			v.Close()
		}
	}()
}

// start new goroutine to run pfn.
func (v *WorkerGroup) ForkGoroutine(pfn func()) {
	go func() {
		v.wait.Add(1)
		defer v.wait.Done()
		defer v.Close()

		pfn()
	}()
}

// notify the group of worker to quit.
func (v *WorkerGroup) Close() error {
	v.closed = true

	select {
	case v.closing <- true:
	default:
	}

	return nil
}

// whether worker group closed.
func (v *WorkerGroup) Closed() bool {
	return v.closed
}

// wait for closing signal, do cleanup and wait for all worker to quit.
func (v *WorkerGroup) Wait(cleanup func()) {
	<-v.closing
	v.closing <- true

	cleanup()

	v.wait.Wait()
}
