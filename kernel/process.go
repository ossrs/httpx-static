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
 This is the process pool for oryx.
*/
package kernel

import (
	"fmt"
	ol "github.com/ossrs/go-oryx-lib/logger"
	"io"
	"os/exec"
	"sync"
)

// The dead body of process.
type TerminatedProcess struct {
	// The process command object.
	Process *exec.Cmd
	// The error got by wait.
	WaitError error
}

// The pool for process, when user create pool, user can exec processes,
// wait for terminated process and close all.
type ProcessPool struct {
	// Started processes, the cmd.Process never be nil.
	cmds map[int]*exec.Cmd
	// Dead processes, get by Wait().
	exitedProcesses chan *TerminatedProcess
	processLock     *sync.Mutex
	// Wait for all processes to quit.
	wait *sync.WaitGroup
	// When closing, user should never care about processes,
	// because all of them will be killed.
	closing chan bool
	// the last context to start command.
	ctx    ol.Context
	closed bool
}

func NewProcessPool() *ProcessPool {
	return &ProcessPool{
		wait:            &sync.WaitGroup{},
		exitedProcesses: make(chan *TerminatedProcess),
		processLock:     &sync.Mutex{},
		cmds:            make(map[int]*exec.Cmd),
		closing:         make(chan bool, 1),
	}
}

// Start new process, user can start many processes.
func (v *ProcessPool) Start(ctx ol.Context, name string, arg ...string) (c *exec.Cmd, err error) {
	v.ctx = ctx

	// create command and start process.
	var cmd *exec.Cmd = exec.Command(name, arg...)

	if err = cmd.Start(); err != nil {
		ol.E(ctx, "start", name, arg, "failed, err is", err)
		return
	}
	pid := cmd.Process.Pid

	func() {
		v.processLock.Lock()
		defer v.processLock.Unlock()
		v.wait.Add(1)
		v.cmds[pid] = cmd
	}()

	// use goroutine to wait for process to quit.
	go func() {
		var err error

		defer func() {
			v.processLock.Lock()
			defer v.processLock.Unlock()
			delete(v.cmds, pid)
			v.wait.Done()
		}()

		defer func() {
			if err == io.EOF {
				return
			}
			pdb := &TerminatedProcess{Process: cmd, WaitError: err}
			select {
			case v.exitedProcesses <- pdb:
			case <-v.closing:
				v.closing <- true
				ol.I(ctx, fmt.Sprintf("drop info for process %v, err is %v", pid, err))
			}
		}()

		// err means process exit failed or other error.
		if err = cmd.Wait(); err != nil {
			select {
			case c := <-v.closing:
				v.closing <- c
				err = io.EOF
			default:
				ol.E(ctx, "process", pid, "exited, err is", err)
			}
			return
		}
	}()

	return cmd, nil
}

// Wait for a dead process.
// @return error io.EOF when pool closed.
// @remark the error may indicates the process not terminate success, user must handle it.
func (v *ProcessPool) Wait() (p *exec.Cmd, err error) {
	select {
	case process, ok := <-v.exitedProcesses:
		if !ok {
			return nil, io.EOF
		}
		return process.Process, process.WaitError
	case c := <-v.closing:
		v.closing <- c
		return nil, io.EOF
	}
}

// io.Closer
// User should reuse the closed process pool.
func (v *ProcessPool) Close() (err error) {
	if v.closed {
		return
	}
	v.closed = true

	ctx := v.ctx

	// notify we are closing, process should drop any info and quit.
	select {
	case v.closing <- true:
	default:
	}

	func() {
		v.processLock.Lock()
		defer v.processLock.Unlock()

		// notify all alive processes to quit.
		for pid, p := range v.cmds {
			var r0 error
			if r0 = p.Process.Kill(); r0 == nil {
				continue
			}
			if err == nil {
				err = r0
			}

			format := "kill process %v failed, r0 is %v, err is %v"
			ol.E(ctx, fmt.Sprintf(format, pid, r0, err))
		}
	}()

	// wait for all processes to quit.
	v.wait.Wait()

	// unblock the waiter.
	// @remark we can close it because no process will write it.
	close(v.exitedProcesses)

	ol.T(ctx, "process pool closed")
	return
}
