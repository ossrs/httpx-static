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
 This the main entrance of shell, to start other processes.
*/
package main

import (
	"encoding/json"
	"fmt"
	oj "github.com/ossrs/go-oryx-lib/json"
	ol "github.com/ossrs/go-oryx-lib/logger"
	oo "github.com/ossrs/go-oryx-lib/options"
	"github.com/ossrs/go-oryx/kernel"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var signature = fmt.Sprintf("SHELL/%v", kernel.Version())

// The service provider.
type ServiceProvider string

func (v ServiceProvider) IsSrs() bool {
	return v == "srs"
}

// The config object for shell module.
type ShellConfig struct {
	kernel.Config
	Rtmplb struct {
		Enabled bool   `json:"enabled"`
		Binary  string `json:"binary"`
		Config  string `json:"config"`
	} `json:"rtmplb"`
	Httplb struct {
		Enabled bool   `json:"enabled"`
		Binary  string `json:"binary"`
		Config  string `json:"config"`
	} `json:"httplb"`
	Worker struct {
		Enabled  bool            `json:"enabled"`
		Provider ServiceProvider `json:"provider"`
		Binary   string          `json:"binary"`
		Config   string          `json:"config"`
		WorkDir  string          `json:"work_dir"`
		Ports    struct {
			Start int `json:"start"`
			Stop  int `json:"stop"`
		} `json:"ports"`
		Service interface{} `json:"service"`
	} `json:"worker"`
}

func (v *ShellConfig) String() string {
	r := &v.Rtmplb
	return fmt.Sprintf("%v rtmplb(enabled=%v,binary=%v,config=%v)", &v.Config, r.Enabled, r.Binary, r.Config)
}

// nil if not srs config.
func (v *ShellConfig) SrsConfig() *SrsServiceConfig {
	r := v.Worker.Service

	if !v.Worker.Provider.IsSrs() {
		return nil
	}

	if r, ok := r.(*SrsServiceConfig); !ok {
		return nil
	} else {
		return r
	}
}

func (v *ShellConfig) Loads(c string) (err error) {
	f := func(c string) (err error) {
		var f *os.File
		if f, err = os.Open(c); err != nil {
			ol.E(nil, "Open config failed, err is", err)
			return
		}
		defer f.Close()

		r := json.NewDecoder(oj.NewJsonPlusReader(f))
		if err = r.Decode(v); err != nil {
			ol.E(nil, "Decode config failed, err is", err)
			return
		}

		return
	}

	// Parse basic config and provider.
	if err = f(c); err != nil {
		return
	}

	if v.Worker.Enabled {
		if !v.Worker.Provider.IsSrs() {
			return fmt.Errorf("Provider(%v) must be srs", string(v.Worker.Provider))
		}

		// Parse srs provider again.
		if v.Worker.Provider.IsSrs() {
			v.Worker.Service = &SrsServiceConfig{}
			if err = f(c); err != nil {
				return
			}
		}
	}

	if err = v.Config.OpenLogger(); err != nil {
		ol.E(nil, "Open logger failed, err is", err)
		return
	}

	if r := &v.Rtmplb; r.Enabled {
		if len(r.Binary) == 0 {
			return fmt.Errorf("Empty rtmplb binary")
		}
		if _, err = exec.LookPath(r.Binary); err != nil {
			ol.E(nil, fmt.Sprintf("Invalid rtmplb binary=%v, err is %v", r.Binary, err))
			return
		}
		if _, err = os.Lstat(r.Config); err != nil {
			ol.E(nil, fmt.Sprintf("Invalid rtmplb config=%v, err is %v", r.Config, err))
			return
		}
	}

	if r := &v.Httplb; r.Enabled {
		if len(r.Binary) == 0 {
			return fmt.Errorf("Empty httplb binary")
		}
		if _, err = exec.LookPath(r.Binary); err != nil {
			ol.E(nil, fmt.Sprintf("Invalid httplb binary=%v, err is %v", r.Binary, err))
			return
		}
		if _, err = os.Lstat(r.Config); err != nil {
			ol.E(nil, fmt.Sprintf("Invalid httplb config=%v, err is %v", r.Config, err))
			return
		}
	}

	if r := &v.Worker; r.Enabled {
		if len(r.Binary) == 0 {
			return fmt.Errorf("Empty worker binary")
		}
		if _, err = exec.LookPath(r.Binary); err != nil {
			ol.E(nil, fmt.Sprintf("Invalid worker binary=%v, err is %v", r.Binary, err))
			return
		}
		if _, err = os.Lstat(r.Config); err != nil {
			ol.E(nil, fmt.Sprintf("Invalid worker config=%v, err is %v", r.Config, err))
			return
		}

		if fi, err := os.Lstat(r.WorkDir); err != nil {
			if !os.IsNotExist(err) {
				ol.E(nil, fmt.Sprintf("Invalid worker dir=%v, err is %v", r.WorkDir, err))
				return err
			} else if err = os.MkdirAll(r.WorkDir, 0755); err != nil {
				ol.E(nil, fmt.Sprintf("Create worker dir=%v failed, err is %v", r.WorkDir, err))
				return err
			}
		} else if !fi.IsDir() {
			return fmt.Errorf("Work dir=%v is not dir", r.WorkDir)
		}

		if r.Ports.Start <= 0 || r.Ports.Stop <= 0 {
			return fmt.Errorf("Ports zone [%v, %v] invalid", r.Ports.Start, r.Ports.Stop)
		}
		if r.Ports.Start >= r.Ports.Stop {
			return fmt.Errorf("Ports zone start=%v should greater than stop=%v", r.Ports.Start, r.Ports.Stop)
		}

		if s := v.SrsConfig(); s != nil {
			if err = s.Check(); err != nil {
				ol.E(nil, "Check srs config failed, err is", err)
				return
			}
		}
	}

	return
}

// The port pool manage available ports.
type PortPool struct {
	shell *ShellBoss
	// alloc new port from ports, fill ports from left,
	// release to ports when free port.
	ports []int
	left  []int
}

// alloc port in [start,stop]
func NewPortPool(start, stop int) *PortPool {
	v := &PortPool{}
	for i := start; i <= stop; i++ {
		if len(v.ports) < 64 {
			v.ports = append(v.ports, i)
		} else {
			v.left = append(v.left, i)
		}
	}
	return v
}

func (v *PortPool) Alloc(nbPort int) (ports []int, err error) {
	if nbPort <= 0 {
		return nil, fmt.Errorf("invalid ports %v", nbPort)
	}
	if len(v.ports)+len(v.left) < nbPort {
		return nil, fmt.Errorf("no %v port available, left %v", nbPort, len(v.ports)+len(v.left))
	}

	if len(v.ports) < nbPort {
		cp := nbPort - len(v.ports)
		v.ports = append(v.ports, v.left[0:cp]...)
		v.left = v.left[cp:]
	}

	ports = v.ports[0:nbPort]
	v.ports = v.ports[nbPort:]
	return
}

func (v *PortPool) Free(port int) {
	v.ports = append(v.ports, port)
}

// The shell to exec all processes.
type ShellBoss struct {
	conf    *ShellConfig
	rtmplb  *exec.Cmd
	httplb  *exec.Cmd
	ctx     ol.Context
	pool    *kernel.ProcessPool
	ports   *PortPool
	workers []*SrsWorker
}

func NewShellBoss(conf *ShellConfig) *ShellBoss {
	v := &ShellBoss{
		conf:    conf,
		ctx:     &kernel.Context{},
		pool:    kernel.NewProcessPool(),
		workers: make([]*SrsWorker, 0),
	}

	c := &v.conf.Worker.Ports
	v.ports = NewPortPool(c.Start, c.Stop)
	return v
}

func (v *ShellBoss) Close() (err error) {
	for _, w := range v.workers {
		w.Close()
	}

	return v.pool.Close()
}

func (v *ShellBoss) ExecBuddies() (err error) {
	ctx := v.ctx

	if r := &v.conf.Rtmplb; r.Enabled {
		if v.rtmplb, err = v.pool.Start(r.Binary, "-c", r.Config); err != nil {
			ol.E(ctx, "Shell: exec rtmplb failed, err is", err)
			return
		}
		p := v.rtmplb
		ol.T(ctx, fmt.Sprintf("Shell: exec rtmplb ok, args=%v, pid=%v", p.Args, p.Process.Pid))
	}

	if r := &v.conf.Httplb; r.Enabled {
		if v.httplb, err = v.pool.Start(r.Binary, "-c", r.Config); err != nil {
			ol.E(ctx, "Shell: exec httplb failed, err is", err)
			return
		}
		p := v.httplb
		ol.T(ctx, fmt.Sprintf("Shell: exec httplb ok, args=%v, pid=%v", p.Args, p.Process.Pid))
	}

	if err = v.execWorker(); err != nil {
		return
	}

	return
}

// Cycle all processes util quit.
func (v *ShellBoss) Cycle() {
	ctx := v.ctx

	for {
		var err error
		var process *exec.Cmd
		if process, err = v.pool.Wait(); err != nil {
			ol.W(ctx, "Shell: wait process failed, err is", err)
			return
		}

		// ignore events when pool closed
		if v.pool.Closed() {
			ol.E(ctx, "Shell: pool terminated")
			return
		}

		// when kernel object exited, close pool
		if process == v.rtmplb || process == v.httplb {
			ol.E(ctx, "Shell: kernel process", process.Process.Pid, "quit, shell quit.")
			v.pool.Close()
			return
		}

		// remove workers
		for i, w := range v.workers {
			if w.process == process {
				w.Close()
				v.workers = append(v.workers[:i], v.workers[i+1:]...)
				break
			}
		}

		// restart worker when terminated.
		if err = v.execWorker(); err != nil {
			ol.E(ctx, "Shell: restart worker failed, err is", err)
			return
		}
	}

	return
}

func (v *ShellBoss) execWorker() (err error) {
	ctx := v.ctx

	r := &v.conf.Worker
	if !r.Enabled {
		return
	}

	if !r.Provider.IsSrs() {
		return
	}

	worker := NewSrsWorker(ctx, v, v.conf.SrsConfig(), v.ports)
	if err = worker.Exec(); err != nil {
		ol.E(ctx, "Shell: start srs worker failed, err is", err)
		return
	}

	v.workers = append(v.workers, worker)
	ol.T(ctx, "Shell: exec worker, total", len(v.workers))
	return
}

func main() {
	var err error

	confFile := oo.ParseArgv("conf/shell.json", kernel.Version(), signature)
	fmt.Println("SHELL is the process forker, config is", confFile)

	conf := &ShellConfig{}
	if err = conf.Loads(confFile); err != nil {
		fmt.Println("Loads config failed, err is", err)
		return
	}
	defer conf.Close()

	ctx := &kernel.Context{}
	ol.T(ctx, fmt.Sprintf("Config ok, %v", conf))

	shell := NewShellBoss(conf)
	f := func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		for s := range c {
			ol.W(ctx, "Shell: got signal", s)
			shell.Close()
			return
		}

	}
	func() {
		if err = shell.ExecBuddies(); err != nil {
			ol.E(ctx, "Shell exec buddies failed, err is", err)
			return
		}
		defer shell.Close()

		go f()

		shell.Cycle()
	}()

	ol.T(ctx, "Shell: terminated")
	return
}
