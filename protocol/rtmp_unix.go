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

// +build darwin dragonfly freebsd nacl netbsd openbsd solaris linux

package protocol

import (
	"github.com/ossrs/go-oryx/core"
	//"net"
)

func (v *RtmpStack) fastSendMessages(iovs ...[]byte) (err error) {
	// we can force to not use writev.
	if !core.Conf.Go.Writev {
		return v.slowSendMessages(iovs...)
	}

	// wait for golang to implements the writev.
	// @see https://github.com/golang/go/issues/13451
	// private writev, @see https://github.com/winlinvip/go/pull/1.
	//if c, ok := v.out.(*net.TCPConn); ok {
	//	if _, err = c.Writev(iovs); err != nil {
	//		return
	//	}
	//	return
	//}

	// send by big-buffer or one-by-one
	return v.slowSendMessages(iovs...)
}
