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
	"time"
)

// the goroutine cycle ignore any error.
func ExampleWorkerContainer_recoverable() {
	var wc WorkerContainer
	wc.GFork("myservice", func(wc WorkerContainer) {
		for {
			select {
			case <-time.After(3 * time.Second):
				// select other channel, do something cycle to get error.
				if err := error(nil); err != nil {
					// recoverable error, log it only and continue or return.
					continue
				}
			case <-wc.QC():
				// when got a quit signal, break the loop.
				// and must notify the container again for other workers
				// in container to quit.
				wc.Quit()
				return
			}
		}
	})
}

// the goroutine cycle absolutely safe, no panic no error to quit.
func ExampleWorkerContainer_safe() {
	var wc WorkerContainer
	wc.GFork("myservice", func(wc WorkerContainer) {
		defer func() {
			if r := recover(); r != nil {
				// log the r and ignore.
				return
			}
		}()

		for {
			select {
			case <-time.After(3 * time.Second):
				// select other channel, do something cycle to get error.
				if err := error(nil); err != nil {
					// recoverable error, log it only and continue or return.
					continue
				}
			case <-wc.QC():
				// when got a quit signal, break the loop.
				// and must notify the container again for other workers
				// in container to quit.
				wc.Quit()
				return
			}
		}
	})
}

// the goroutine cycle notify container to quit when error.
func ExampleWorkerContainer_fatal() {
	var wc WorkerContainer
	wc.GFork("myservice", func(wc WorkerContainer) {
		for {
			select {
			case <-time.After(3 * time.Second):
				// select other channel, do something cycle to get error.
				if err := error(nil); err != nil {
					// when got none-recoverable error, notify container to quit.
					wc.Quit()
					return
				}
			case <-wc.QC():
				// when got a quit signal, break the loop.
				// and must notify the container again for other workers
				// in container to quit.
				wc.Quit()
				return
			}
		}
	})
}
