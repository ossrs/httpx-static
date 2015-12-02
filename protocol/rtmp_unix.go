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
	"io"
	"net"
	"os"
	"reflect"
	"sync"
	"syscall"
	"unsafe"
)

type RtmpSysFd struct {
	// system fd, from (c *TCPConn).(fd *netFD).sysfd
	sysfd uintptr
	// the *netFD, from (c *TCPConn).fd
	fd reflect.Value
	// the pollDesc, from (c *TCPConn).(fd *netFD).pd
	pd reflect.Value

	// locker.
	lock sync.Mutex

	// rollback to slow send.
	rtmp *RtmpStack
	// whether ok to writev.
	ok bool
}

func (v *RtmpSysFd) init() interface{} {
	// use writev when got fd.
	// @see https://github.com/winlinvip/vectorio/blob/master/vectorio.go
	var ok bool
	var c *net.TCPConn
	if c, ok = v.rtmp.out.(*net.TCPConn); !ok {
		return v
	}

	// get c which is net.TCPConn
	var fc reflect.Value
	if fc = reflect.ValueOf(c); fc.Kind() == reflect.Ptr {
		fc = fc.Elem()
	}

	// get the ptr.
	v.fd = fc.FieldByName("fd")

	// get c.fd which is net.netFD, in net/net.go
	var ffd reflect.Value = v.fd
	if ffd.Kind() == reflect.Ptr {
		ffd = ffd.Elem()
	}

	// get the ptr.
	if v.pd = ffd.FieldByName("pd"); v.pd.Kind() != reflect.Ptr {
		v.pd = v.pd.Addr()
	}

	// get c.fd.pd which is pollDesc, in net/fd_poll_runtime.go
	var fpd reflect.Value = v.pd
	if fpd.Kind() == reflect.Ptr {
		fpd = fpd.Elem()
	}

	// get c.fd.sysfd which is int, in net/fd_unix.go
	var fsysfd reflect.Value
	if fsysfd = ffd.FieldByName("sysfd"); fsysfd.Kind() == reflect.Ptr {
		fsysfd = fsysfd.Elem()
	}
	v.sysfd = uintptr(fsysfd.Int())

	// fast writev is ok.
	v.ok = true

	return v
}

// delegate v.fd.writeLock
func (v *RtmpSysFd) writeLock() (err error) {
	v.lock.Lock()
	return
}

// delegate v.fd.writeUnlock
func (v *RtmpSysFd) writeUnlock() {
	v.lock.Unlock()
	return
}

// delegate v.pd.PrepareWrite
func (v *RtmpSysFd) PrepareWrite() (err error) {
	return
}

// delegate v.pd.WaitWrite
func (v *RtmpSysFd) WaitWrite() (err error) {
	// error: reflect.Value.Call using value obtained using unexported field
	return
}

func (v *RtmpSysFd) writev(iovs ...[]byte) (err error) {
	if !v.ok {
		return v.rtmp.slowSendMessages(iovs...)
	}

	// lock the fd.
	if err = v.writeLock(); err != nil {
		return
	}
	defer v.writeUnlock()
	if err = v.PrepareWrite(); err != nil {
		return
	}

	// prepare data.
	var total int
	iovecs := make([]syscall.Iovec, len(iovs))
	for i, iov := range iovs {
		total += len(iov)
		iovecs[i] = syscall.Iovec{&iov[0], uint64(len(iov))}
	}

	var nn int
	for {
		var n int
		if n, err = writev(v.sysfd, iovecs); err != nil {
			return
		}
		// TODO: FIXME: implements is.
		if n != total {
			core.Error.Println("fatal.")
			panic(fmt.Sprintf("writev n=%v, total=%v", n, total))
		}

		if n > 0 {
			nn += n
		}
		if nn == total {
			break
		}
		if err == syscall.EAGAIN {
			if err = v.WaitWrite(); err == nil {
				continue
			}
		}
		if err != nil {
			break
		}
		if n == 0 {
			err = io.ErrUnexpectedEOF
			break
		}
	}

	if _, ok := err.(syscall.Errno); ok {
		err = os.NewSyscallError("writev", err)
	}

	return
}

func (v *RtmpStack) fastSendMessages(iovs ...[]byte) (err error) {
	// initialize the fd.
	if v.sysfd == nil {
		fd := &RtmpSysFd{
			rtmp: v,
		}
		v.sysfd = fd.init()
	}

	if v, ok := v.sysfd.(*RtmpSysFd); ok {
		return v.writev(iovs...)
	}

	return v.slowSendMessages(iovs...)
}

func writev(fd uintptr, iovs []syscall.Iovec) (int, error) {
	iovsPtr := uintptr(unsafe.Pointer(&iovs[0]))
	iovsLen := uintptr(len(iovs))

	n, _, e0 := syscall.Syscall(syscall.SYS_WRITEV, fd, iovsPtr, iovsLen)

	if e0 != 0 {
		return 0, syscall.Errno(e0)
	}

	return int(n), nil
}
