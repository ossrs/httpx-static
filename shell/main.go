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
)

var signature = fmt.Sprintf("RTMPLB/%v", kernel.Version())

// The config object for shell module.
type ShellConfig struct {
	kernel.Config
	Rtmplb struct {
		Enabled bool   `json:"enabled"`
		Binary  string `json:"binary"`
		Config  string `json:"config"`
		// The binary to exec.
		binary string
	} `json:"rtmplb"`
}

func (v *ShellConfig) String() string {
	r := &v.Rtmplb
	return fmt.Sprintf("%v rtmplb(enabled=%v,binary=%v,config=%v)", &v.Config, r.Enabled, r.Binary, r.Config)
}

func (v *ShellConfig) Loads(c string) (err error) {
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

	if err = v.Config.OpenLogger(); err != nil {
		ol.E(nil, "Open logger failed, err is", err)
		return
	}

	if r := &v.Rtmplb; r.Enabled {
		if len(r.Binary) == 0 {
			return fmt.Errorf("Empty rtmplb binary")
		}
		if r.binary, err = exec.LookPath(r.Binary); err != nil {
			ol.E(nil, fmt.Sprintf("Invalid rtmplb binary=%v, err is %v", r.Binary, err))
			return
		}
		if _, err = os.Lstat(r.Config); err != nil {
			ol.E(nil, fmt.Sprintf("Invalid rtmplb config=%v, err is %v", r.Config, err))
			return
		}
	}

	return
}

// The shell to exec all processes.
type ShellBoss struct {
	conf   *ShellConfig
	rtmplb *exec.Cmd
	ctx    ol.Context
}

func NewShellBoss(conf *ShellConfig) *ShellBoss {
	return &ShellBoss{
		conf: conf,
		ctx:  &kernel.Context{},
	}
}

func (v *ShellBoss) Close() (err error) {
	ctx := v.ctx

	if v.conf.Rtmplb.Enabled {
		if r := v.rtmplb.ProcessState; r == nil || !r.Exited() {
			var r0, r1 error

			if r0 = v.rtmplb.Process.Kill(); err == nil {
				err = r0
			}

			if _, r1 = v.rtmplb.Process.Wait(); err == nil {
				err = r1
			}

			ol.W(ctx, fmt.Sprintf("Shell: rtmplb exited, r0=%v, r1=%v, err is %v", r0, r1, err))
		} else {
			ol.T(ctx, "Shell: rtmplb already exited")
		}
	}

	return
}

func (v *ShellBoss) ExecBuddies() (err error) {
	ctx := v.ctx

	if r := &v.conf.Rtmplb; r.Enabled {
		v.rtmplb = exec.Command(r.binary, "-c", r.Config)
		if err = v.rtmplb.Start(); err != nil {
			ol.E(ctx, "Shell: exec rtmplb failed, err is", err)
			return
		}

		p := v.rtmplb
		ol.T(ctx, fmt.Sprintf("Shell: exec rtmplb ok, args=%v, pid=%v", p.Args, p.Process.Pid))
	}
	return
}

func (v *ShellBoss) WaitRtmplb() (err error) {
	ctx := v.ctx

	if !v.conf.Rtmplb.Enabled {
		return
	}

	if err = v.rtmplb.Wait(); err != nil {
		ol.E(ctx, "Shell: rtmplb failed, err is", err)
		return
	}

	err = fmt.Errorf("rtmplb exited")
	ol.E(ctx, fmt.Sprintf("Shell: rtmplb exited, err is", err))

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

	if err = shell.WaitRtmplb(); err != nil {
		ol.E(ctx, "Shell: rtmplb failed, err is", err)
		return
	}

	return
}
