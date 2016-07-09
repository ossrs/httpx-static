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

import (
	"errors"
	"io"
	"net"
)

// ErrQuit used for goroutine to return.
var ErrQuit = errors.New("system quit")

// ErrOverflow when channel overflow, for example, the c0c1 never overflow
// when channel buffer size set to 2.
var ErrOverflow = errors.New("system overflow")

// ErrTimeout when io timeout to wait.
var ErrTimeout = errors.New("io timeout")

// ErrVhostNotFound when the rtmp vhost not found.
var ErrVhostNotFound = errors.New("vhost not found")

// IsNormalQuit whether the object in recover or returned error can ignore,
// for instance, the error is a Quit error.
func IsNormalQuit(err interface{}) bool {
	if err == nil {
		return true
	}

	if err, ok := err.(error); ok {
		// client EOF.
		if err == io.EOF {
			return true
		}

		// manual quit or read timeout.
		if err == ErrQuit || err == ErrTimeout {
			return true
		}

		// network timeout.
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return true
		}
	}

	return false
}
