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

package core

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
	// @remark this quit always return a core.QuitError error, which can be ignore.
	Quit() (err error)
	// fork a new goroutine with work container.
	// the param f can be a global func or object method.
	// the param name is the goroutine name.
	GFork(name string, f func(WorkerContainer))
}

// which used for quit.
// TODO: FIXME: server should use it.
type Quiter struct {
	closing chan bool
}

func NewQuiter() *Quiter {
	return &Quiter{
		closing: make(chan bool, 1),
	}
}

func (v *Quiter) QC() <-chan bool {
	return v.closing
}

func (v *Quiter) Quit() (err error) {
	select {
	case v.closing <- true:
	default:
	}
	return QuitError
}
