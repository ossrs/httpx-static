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
 The example for kernel.
*/
package kernel_test

import (
	oa "github.com/ossrs/go-oryx-lib/asprocess"
	ol "github.com/ossrs/go-oryx-lib/logger"
	"github.com/ossrs/go-oryx/kernel"
	"io"
	"net"
	"os/exec"
	"syscall"
	"time"
)

func ExampleWorkerGroup() {
	// other goroutine to notify worker group to quit.
	closing := make(chan bool, 1)
	// for example, user can use asprocess to watch without exit to write closing.
	oa.WatchNoExit(nil, oa.Interval, closing)

	// use group to sync workers.
	wg := kernel.NewWorkerGroup()
	defer wg.Close()

	// quit for external events.
	wg.QuitForChan(closing)
	// quit for signals.
	wg.QuitForSignals(nil, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	// start goroutine to worker, quit when worker quit.
	// for example, the listener for http.
	var listener net.Listener
	wg.ForkGoroutine(func() {
		for {
			if conn, err := listener.Accept(); err != nil {
				if err != io.EOF {
					// log error.
				}
				return
			} else {
				// serve conn

				defer conn.Close()
			}
		}
	}, func() {
		listener.Close()
	})

	// start goroutine without cleanup.
	var closed bool
	wg.ForkGoroutine(func() {
		for !closed {
			// do something util quit.
			time.Sleep(time.Duration(1) * time.Second)
		}
	}, func() {
		closed = true
	})

	// wait for quit.
	wg.Wait()
}

func ExampleProcessPool() {
	pp := kernel.NewProcessPool()
	defer pp.Close()

	// start a worker.
	var err error
	var cmd *exec.Cmd
	if cmd, err = pp.Start(nil, "ls", "-al"); err != nil {
		return
	}

	// start other workers.

	// wait for some processes to quit.
	var c *exec.Cmd
	if c, err = pp.Wait(); err != nil {
		return
	}

	// do something for the terminated process.
	if cmd == c {
		// cmd must be c.
	}
}

func ExampleTcpListeners() {
	// create listener by addresses
	ls, err := kernel.NewTcpListeners([]string{"tcp://:1935", "tcp://:1985", "tcp://:8080"})
	if err != nil {
		return
	}
	defer ls.Close()

	// listen all addresses
	if err = ls.ListenTCP(); err != nil {
		return
	}

	// accept conn from any listeners
	for {
		conn, err := ls.AcceptTCP()
		if err != nil {
			return
		}

		// serve and close conn
		defer conn.Close()
	}
}

func ExampleContext() {
	ctx := &kernel.Context{}
	ol.T(ctx, "create context ok")
}
