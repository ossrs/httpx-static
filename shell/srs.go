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

// srs worker for shell.
package main

import (
	"fmt"
	ol "github.com/ossrs/go-oryx-lib/logger"
	"os/exec"
)

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

func (v *SrsServiceConfig) Check() (err error) {
	if len(v.BigBinary) == 0 {
		return fmt.Errorf("Empty big binary")
	} else if len(v.Variables.BigBinary) == 0 {
		return fmt.Errorf("Empty variable big binary")
	} else if len(v.Variables.ApiPort) == 0 {
		return fmt.Errorf("Empty variable api port")
	} else if len(v.Variables.BigPort) == 0 {
		return fmt.Errorf("Empty variable big port")
	} else if len(v.Variables.HttpPort) == 0 {
		return fmt.Errorf("Empty variable http port")
	} else if len(v.Variables.PidFile) == 0 {
		return fmt.Errorf("Empty variable pid file")
	} else if len(v.Variables.RtmpPort) == 0 {
		return fmt.Errorf("Empty variable rtmp port")
	} else if len(v.Variables.WorkDir) == 0 {
		return fmt.Errorf("Empty variable work dir")
	}

	return
}

// The srs stream worker.
type SrsWorker struct {
	shell   *ShellBoss
	process *exec.Cmd
	conf    *SrsServiceConfig
	ctx     ol.Context
}

func NewSrsWorker(ctx ol.Context, shell *ShellBoss, conf *SrsServiceConfig) *SrsWorker {
	v := &SrsWorker{ctx: ctx, shell: shell, conf: conf}
	return v
}

func (v *SrsWorker) Exec(ports *PortPool) (err error) {
	return
	ctx := v.ctx
	r := v.shell.conf.Worker

	if v.process, err = v.shell.pool.Start(r.Binary, "-c", r.Config); err != nil {
		ol.E(ctx, "exec worker failed, err is", err)
		return
	}

	p := v.process
	ol.T(ctx, fmt.Sprintf("exec worker ok, args=%v, pid=%v", p.Args, p.Process.Pid))

	return
}
