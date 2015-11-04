// The MIT License (MIT)
//
// Copyright (c) 2013-2015 SRS(simple-rtmp-server)
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

// Unix to run in daemon.

package app

import (
	"fmt"
	"github.com/simple-rtmp-server/go-srs/core"
	"os"
	"runtime"
	"syscall"
)

func (s *Server) daemon() error {
	// set to single thread mode.
	runtime.GOMAXPROCS(1)

	// lock goroutine to thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// start in daemon if possible.
	pid, r2, err := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)

	if err != 0 {
		return fmt.Errorf("fork failed, err is %v", err)
	}

	if r2 < 0 {
		return fmt.Errorf("fork failed, r2 is", r2)
	}

	// for darwin, the child process forked.
	if runtime.GOOS == "darwin" && r2 == 1 {
		pid = 0
	}

	// exit parent process.
	if pid > 0 {
		os.Exit(0)
	}

	// setsid for child process to daemon.
	if _, err := syscall.Setsid(); err != nil {
		return err
	}

	return nil
}

func (s *Server) daemonOnRunning() {
	core.GsTrace.Println("server run in daemon, pid is", os.Getpid(), "and ppid is", os.Getppid())
}
