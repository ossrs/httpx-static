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
	"reflect"
)

var signature = fmt.Sprintf("SHELL/%v", kernel.Version())

// The service provider.
type ServiceProvider string

func (v ServiceProvider) IsSrs() bool {
	return v == "srs"
}

// Service SRS specified config.
type SrsServiceConfig struct {
	BigBinary string `json:"big_binary"`
	Variables struct {
		RtmpPort  string `json:"rtmp_port"`
		ApiPort   string `json:"api_port"`
		HttpPort  string `json:"http_port"`
		BigPort   string `json:"big_port"`
		BigBinary string `json:"big_binary"`
		PidFile   string `json:"pid_file"`
		WorkDir   string `json:"work_dir"`
	} `json:"variables"`
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

func (v *ShellConfig) SrsConfig() (c *SrsServiceConfig, err error) {
	r := v.Worker.Service

	if r, ok := r.(*SrsServiceConfig); !ok {
		return nil, fmt.Errorf("Config is not srs service, actual is %v", reflect.TypeOf(r))
	} else {
		return r, nil
	}

	return
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

		if r.Provider.IsSrs() {
			if s, err := v.SrsConfig(); err != nil {
				ol.E(nil, "Service srs config invalid, err is", err)
				return err
			} else if len(s.BigBinary) == 0 {
				return fmt.Errorf("Empty big binary")
			} else if len(s.Variables.BigBinary) == 0 {
				return fmt.Errorf("Empty variable big binary")
			} else if len(s.Variables.ApiPort) == 0 {
				return fmt.Errorf("Empty variable api port")
			} else if len(s.Variables.BigPort) == 0 {
				return fmt.Errorf("Empty variable big port")
			} else if len(s.Variables.HttpPort) == 0 {
				return fmt.Errorf("Empty variable http port")
			} else if len(s.Variables.PidFile) == 0 {
				return fmt.Errorf("Empty variable pid file")
			} else if len(s.Variables.RtmpPort) == 0 {
				return fmt.Errorf("Empty variable rtmp port")
			} else if len(s.Variables.WorkDir) == 0 {
				return fmt.Errorf("Empty variable work dir")
			}
		}
	}

	return
}

// The shell to exec all processes.
type ShellBoss struct {
	conf   *ShellConfig
	rtmplb *exec.Cmd
	httplb *exec.Cmd
	ctx    ol.Context
	pool   *kernel.ProcessPool
}

func NewShellBoss(conf *ShellConfig) *ShellBoss {
	v := &ShellBoss{
		conf: conf,
		ctx:  &kernel.Context{},
	}
	v.pool = kernel.NewProcessPool(v.ctx)
	return v
}

func (v *ShellBoss) Close() (err error) {
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

	return
}

func (v *ShellBoss) Wait() {
	ctx := v.ctx

	var err error
	var process *exec.Cmd
	if process, err = v.pool.Wait(); err != nil {
		ol.W(ctx, "Shell: wait process failed, err is", err)
		return
	}

	if process == v.rtmplb || process == v.httplb {
		ol.W(ctx, "Shell: kernel process", process.Process.Pid, "quit, shell quit.")
		return
	}

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
	if err = shell.ExecBuddies(); err != nil {
		ol.E(ctx, "Shell exec buddies failed, err is", err)
		return
	}
	defer shell.Close()

	shell.Wait()

	return
}
