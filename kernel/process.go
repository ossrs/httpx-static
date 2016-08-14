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
	"os/exec"
	"sync"
)

// The dead body of process.
type ProcessDeadBody struct {
	// The process command object.
	Process *exec.Cmd
	// The error got by wait.
	WaitError error
}

// When pool disposed.
var PoolDisposed = fmt.Errorf("pool is disposed")

// The pool for process, when user create pool, user can exec processes,
// wait for terminated process and close all.
type ProcessPool struct {
	ctx ol.Context
	// Alive processes.
	processes map[int]*exec.Cmd
	// Dead processes, get by Wait().
	exitedProcesses chan *ProcessDeadBody
	processLock     *sync.Mutex
	// Wait for all processes to quit.
	wait *sync.WaitGroup
	// Whether pool is closed.
	disposed    bool
	disposeLock *sync.Mutex
	// When closing, user should never care about processes,
	// because all of them will be killed.
	closing chan bool
}

func NewProcessPool() *ProcessPool {
	return &ProcessPool{
		wait:            &sync.WaitGroup{},
		exitedProcesses: make(chan *ProcessDeadBody),
		processLock:     &sync.Mutex{},
		processes:       make(map[int]*exec.Cmd),
		disposeLock:     &sync.Mutex{},
		closing:         make(chan bool, 1),
	}
}

// Start new process, user can start many processes.
// @return error PoolDisposed when pool disposed.
func (v *ProcessPool) Start(ctx ol.Context, name string, arg ...string) (c *exec.Cmd, err error) {
	// when disposed, should never use it again.
	v.disposeLock.Lock()
	defer v.disposeLock.Unlock()
	if v.disposed {
		return nil, PoolDisposed
	}

	// create command and start process.
	var process *exec.Cmd = exec.Command(name, arg...)

	if err = process.Start(); err != nil {
		ol.E(ctx, "start", name, arg, "failed, err is", err)
		return
	}
	pid := process.Process.Pid

	func() {
		v.processLock.Lock()
		defer v.processLock.Unlock()
		v.wait.Add(1)
		v.processes[pid] = process
	}()

	// use goroutine to wait for process to quit.
	go func() {
		var err error

		defer func() {
			v.processLock.Lock()
			defer v.processLock.Unlock()
			delete(v.processes, pid)
			v.wait.Done()
		}()

		defer func() {
			pdb := &ProcessDeadBody{Process: process, WaitError: err}
			select {
			case v.exitedProcesses <- pdb:
			case <-v.closing:
				v.closing <- true
				ol.I(ctx, fmt.Sprintf("drop info for process %v, err is %v", pid, err))
			}
		}()

		if err = process.Wait(); err != nil {
			if !v.disposed {
				ol.E(ctx, "process", pid, "exited, err is", err)
			}
			return
		}
	}()

	return process, nil
}

// Wait for a dead process.
// @return error PoolDisposed when pool disposed.
func (v *ProcessPool) Wait() (p *exec.Cmd, err error) {
	// should never lock, for it's wait gotoutine.
	if err = func() error {
		// when disposed, should never use it again.
		v.disposeLock.Lock()
		defer v.disposeLock.Unlock()
		if v.disposed {
			return PoolDisposed
		}
		return nil
	}(); err != nil {
		return
	}

	select {
	case process, ok := <-v.exitedProcesses:
		if !ok {
			return nil, PoolDisposed
		}
		return process.Process, process.WaitError
	case c := <-v.closing:
		v.closing <- c
		return nil, PoolDisposed
	}
}

// interface io.Closer
// @return error PoolDisposed when pool disposed.
func (v *ProcessPool) Close(ctx ol.Context) (err error) {
	// notify we are closing, process should drop any info and quit.
	select {
	case v.closing <- true:
	default:
	}

	// when disposed, should never dispose again.
	v.disposeLock.Lock()
	defer v.disposeLock.Unlock()
	if v.disposed {
		return PoolDisposed
	}
	v.disposed = true

	func() {
		v.processLock.Lock()
		defer v.processLock.Unlock()

		// notify all alive processes to quit.
		for pid, p := range v.processes {
			if r0 := p.Process.Kill(); r0 != nil {
				if err == nil {
					err = r0
				}
				ol.E(ctx, fmt.Sprintf("kill process %v failed, r0 is %v, err is %v", pid, r0, err))
			}
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

// whether current pool closed.
func (v *ProcessPool) Closed() bool {
	v.disposeLock.Lock()
	defer v.disposeLock.Unlock()
	return v.disposed
}
