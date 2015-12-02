// The MIT License (MIT)
//
// Copyright (c) 2013-2015 Oryx(ossrs)
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
	"fmt"
	"github.com/ossrs/go-oryx/core"
	"net"
	"reflect"
	"syscall"
	"unsafe"
)

func (v *RtmpStack) fastSendMessages(iovs ...[]byte) (err error) {
	// initialize the fd.
	if v.fd == 0 {
		var ok bool
		var c *net.TCPConn
		if c, ok = v.out.(*net.TCPConn); !ok {
			return v.slowSendMessages(iovs...)
		}

		var vfd reflect.Value
		// get c which is net.TCPConn
		if vfd = reflect.ValueOf(c); vfd.Kind() == reflect.Ptr {
			vfd = vfd.Elem()
		}
		// get c.fd which is net.netFD, in net/net.go
		if vfd = vfd.FieldByName("fd"); vfd.Kind() == reflect.Ptr {
			vfd = vfd.Elem()
		}
		// get c.fd.sysfd which is int, in net/fd_unix.go
		if vfd = vfd.FieldByName("sysfd"); vfd.Kind() == reflect.Ptr {
			vfd = vfd.Elem()
		}
		// get fd value.
		v.fd = vfd.Int()
	}

	// use writev when got fd.
	// @see https://github.com/winlinvip/vectorio/blob/master/vectorio.go
	if v.fd > 0 {
		iovecs := make([]syscall.Iovec, len(iovs))
		for i, iov := range iovs {
			iovecs[i] = syscall.Iovec{&iov[0], uint64(len(iov))}
		}

		if _, err = writev(uintptr(v.fd), iovecs); err != nil {
			return
		}
		core.Trace.Println("ok")
		return
	}

	return v.slowSendMessages(iovs...)
}

func writev(fd uintptr, iovs []syscall.Iovec) (int, error) {
	iovsPtr := uintptr(unsafe.Pointer(&iovs[0]))
	iovsLen := uintptr(len(iovs))

	n, _, errno := syscall.Syscall(syscall.SYS_WRITEV, fd, iovsPtr, iovsLen)

	if errno != 0 {
		return 0, fmt.Errorf("writev failed, errno=%v", int64(errno))
	}

	return int(n), nil
}
