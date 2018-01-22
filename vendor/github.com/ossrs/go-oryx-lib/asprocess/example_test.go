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

package asprocess_test

import (
	oa "github.com/ossrs/go-oryx-lib/asprocess"
	"os"
	"os/signal"
	"syscall"
)

func ExampleAsProcess() {
	// Without context and callback.
	oa.Watch(nil, oa.CheckParentInterval, nil)

	// Without context, use callback to cleanup.
	oa.Watch(nil, oa.CheckParentInterval, oa.Cleanup(func() {
		// Do cleanup when quit.
	}))
}

func ExampleAsProcess_NoQuit() {
	// User control the quit event.
	q := make(chan bool, 1)
	oa.WatchNoExit(nil, oa.CheckParentInterval, q)

	// Quit when parent changed or signals.
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	select {
	case <-q:
		// Quit for parent changed.
	case <-c:
		// Quit for signal.
	}
}
