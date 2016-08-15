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
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Service SRS specified config.
type SrsServiceConfig struct {
	BigBinary string `json:"big_binary"`
	Variables struct {
		RtmpPort      string `json:"rtmp_port"`
		ApiPort       string `json:"api_port"`
		HttpPort      string `json:"http_port"`
		BigPort       string `json:"big_port"`
		BigBinary     string `json:"big_binary"`
		WorkDir       string `json:"work_dir"`
		HttpProxyPort string `json:"http_proxy_port"`
		BigProxyPort  string `json:"big_proxy_port"`
	} `json:"variables"`
}

func (v *SrsServiceConfig) String() string {
	r := &v.Variables
	return fmt.Sprintf("srs<binary=%v,rtmp=%v,api=%v,http=%v,big=%v,bigbin=%v,dir=%v,phttp=%v,pbig=%v>",
		v.BigBinary, r.RtmpPort, r.ApiPort, r.HttpPort, r.BigPort, r.BigBinary, r.WorkDir, r.HttpProxyPort, r.BigProxyPort)
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
	} else if len(v.Variables.RtmpPort) == 0 {
		return fmt.Errorf("Empty variable rtmp port")
	} else if len(v.Variables.WorkDir) == 0 {
		return fmt.Errorf("Empty variable work dir")
	} else if len(v.Variables.HttpProxyPort) == 0 {
		return fmt.Errorf("Empty variable http proxy port")
	} else if len(v.Variables.BigProxyPort) == 0 {
		return fmt.Errorf("Empty variable big proxy port")
	}

	return
}

// The state of srs worker
type SrsState int

const (
	// worker is init and not start.
	SrsStateInit SrsState = 1 << iota
	// worker is active serving state.
	SrsStateActive
	// worker is deprecated state, should ignore its terminated.
	// for example, when new process change from init to active,
	// the old processes should change to deprecated.
	SrsStateDeprecated
)

func (v SrsState) String() string {
	switch v {
	case SrsStateInit:
		return "init"
	case SrsStateActive:
		return "active"
	case SrsStateDeprecated:
		return "deprecated"
	default:
		return "unknown"
	}
}

// The srs stream worker.
type SrsWorker struct {
	shell   *ShellBoss
	process *exec.Cmd
	conf    *SrsServiceConfig
	ctx     ol.Context
	lock    *sync.Mutex
	// the work dir for this process, a sub folder under config.
	workDir string
	// the config for this process, under workDir.
	config string
	// allocated ports(maybe not used).
	ports []int
	pool  *PortPool
	// the used port.
	rtmp int
	http int
	api  int
	big  int
	// version of worker.
	version *SrsVersion
	// the state of process.
	state SrsState
}

func NewSrsWorker(ctx ol.Context, shell *ShellBoss, conf *SrsServiceConfig, ports *PortPool) *SrsWorker {
	v := &SrsWorker{
		ctx: ctx, shell: shell, conf: conf,
		pool: ports, lock: &sync.Mutex{},
		state: SrsStateInit,
	}
	return v
}

func (v *SrsWorker) String() string {
	return fmt.Sprintf("srs worker, pid=%v, state=%v, version=%v", v.process.Process.Pid, v.state, v.version)
}

func (v *SrsWorker) Close() error {
	v.lock.Lock()
	defer v.lock.Unlock()

	// free the ports.
	for _, p := range v.ports {
		v.pool.Free(p)
	}
	v.ports = nil

	// TODO: FIXME: cleanup workdir.

	return nil
}

func (v *SrsWorker) Exec() (err error) {
	ctx := v.ctx
	r := v.shell.conf.Worker
	s := v.shell.conf.SrsConfig()

	// read config template from file to build the config for srs.
	var configTemplate string
	if f, err := os.Open(r.Config); err != nil {
		ol.E(ctx, "open config", r.Config, "failed, err is", err)
		return err
	} else {
		defer f.Close()
		if b, err := ioutil.ReadAll(f); err != nil {
			ol.E(ctx, "read config failed, err is", err)
			return err
		} else {
			configTemplate = string(b)
		}
	}

	// alloc all ports, althrough maybe not use it.
	if ports, err := v.pool.Alloc(4); err != nil {
		ol.E(ctx, "alloc ports failed, err is", err)
		return err
	} else {
		v.ports = append(v.ports, ports...)
		v.rtmp, v.http, v.api, v.big = ports[0], ports[1], ports[2], ports[3]
	}

	// build all port.
	conf := configTemplate
	conf = strings.Replace(conf, s.Variables.RtmpPort, strconv.Itoa(v.rtmp), -1)
	conf = strings.Replace(conf, s.Variables.HttpPort, strconv.Itoa(v.http), -1)
	conf = strings.Replace(conf, s.Variables.ApiPort, strconv.Itoa(v.api), -1)
	conf = strings.Replace(conf, s.Variables.BigPort, strconv.Itoa(v.big), -1)
	// for http proxy for hls+
	conf = strings.Replace(conf, s.Variables.HttpProxyPort, strconv.Itoa(v.shell.conf.Httplb.Http), -1)
	// for big proxy port.
	conf = strings.Replace(conf, s.Variables.BigProxyPort, strconv.Itoa(v.shell.conf.Apilb.Big), -1)

	// build other variables
	if len(s.BigBinary) > 0 {
		conf = strings.Replace(conf, s.Variables.BigBinary, s.BigBinary, -1)
	}

	v.workDir = path.Join(r.WorkDir, fmt.Sprintf("srs/%v", time.Now().Format("2006-01-02-15:04:05.000")))
	if wd := path.Join(v.workDir, "objs/nginx/html"); true {
		if err = os.MkdirAll(wd, 0755); err != nil {
			ol.E(ctx, "create srs dir", wd, "failed, err is", err)
			return
		}
	}
	conf = strings.Replace(conf, s.Variables.WorkDir, v.workDir, -1)

	// symbol link all binaries to work dir.
	var pwd string
	if pwd, err = os.Getwd(); err != nil {
		ol.E(ctx, "getwd failed, err is", err)
		return
	}
	if bin := s.BigBinary; len(bin) > 0 && !path.IsAbs(bin) {
		from, to := path.Join(pwd, bin), path.Join(v.workDir, bin)
		if err = os.Symlink(from, to); err != nil {
			ol.E(ctx, fmt.Sprintf("symlink %v=%v failed, err is %v", from, to, err))
			return
		}
	}
	if bin := r.Binary; !path.IsAbs(bin) {
		from, to := path.Join(pwd, bin), path.Join(v.workDir, bin)
		if err = os.Symlink(from, to); err != nil {
			ol.E(ctx, fmt.Sprintf("symlink %v=%v failed, err is %v", from, to, err))
			return
		}
	}

	// write to config file.
	v.config = path.Join(v.workDir, "srs.conf")
	if f, err := os.Create(v.config); err != nil {
		ol.E(ctx, "create config failed, err is", err)
		return err
	} else {
		defer f.Close()
		if _, err = f.WriteString(conf); err != nil {
			ol.E(ctx, "write config failed, err is", err)
			return err
		}
	}
	ol.T(ctx, fmt.Sprintf("srs ports(rtmp=%v,http=%v,api=%v,big=%v), cwd=%v, config=%v",
		v.rtmp, v.http, v.api, v.big, v.workDir, v.config))

	if v.process, err = v.shell.pool.Start(ctx, r.Binary, "-c", v.config); err != nil {
		ol.E(ctx, "exec worker failed, err is", err)
		return
	}

	p := v.process
	ol.T(ctx, fmt.Sprintf("exec worker ok, args=%v, pid=%v", p.Args, p.Process.Pid))

	return
}
