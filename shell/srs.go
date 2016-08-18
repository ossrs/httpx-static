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
	BigBinary    string `json:"big_binary"`
	BitchBinary  string `json:"bitch_binary"`
	FfmpegBinary string `json:"ffmpeg_binary"`
	DnsBinary    string `json:"dns_binary"`
	Variables    struct {
		RtmpPort      string `json:"rtmp_port"`
		ApiPort       string `json:"api_port"`
		HttpPort      string `json:"http_port"`
		BigPort       string `json:"big_port"`
		BigBinary     string `json:"big_binary"`
		BitchPort     string `json:"bitch_port"`
		BitchBinary   string `json:"bitch_binary"`
		FfmpegBinary  string `json:"ffmpeg_binary"`
		DnsPort       string `json:"dns_port"`
		DnsBinary     string `json:"dns_binary"`
		WorkDir       string `json:"work_dir"`
		HttpProxyPort string `json:"http_proxy_port"`
		BigProxyPort  string `json:"big_proxy_port"`
	} `json:"variables"`
}

func NewSrsServiceConfig() *SrsServiceConfig {
	return &SrsServiceConfig{}
}

func (v *SrsServiceConfig) String() string {
	r := &v.Variables
	common := fmt.Sprintf("binary=%v,rtmp=%v,api=%v,http=%v",
		v.BigBinary, r.RtmpPort, r.ApiPort, r.HttpPort)
	oaprocess := fmt.Sprintf("big=%v,bigbin=%v,bitch=%v,bitchbin=%v,dns=%v,dnsbin=%v",
		r.BigPort, r.BigBinary, r.BitchPort, r.BitchBinary, r.DnsPort, r.DnsBinary)
	others := fmt.Sprintf("ffmpeg=%v,dir=%v,phttp=%v,pbig=%v",
		r.FfmpegBinary, r.WorkDir, r.HttpProxyPort, r.BigProxyPort)
	return fmt.Sprintf("srs<%v,%v,%v>", common, oaprocess, others)
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
	} else if len(v.Variables.BitchPort) == 0 {
		return fmt.Errorf("Empty variable bitch port")
	} else if len(v.Variables.BitchBinary) == 0 {
		return fmt.Errorf("Empty variable bitch binary")
	} else if len(v.Variables.FfmpegBinary) == 0 {
		return fmt.Errorf("Empty variable ffmpeg binary")
	} else if len(v.Variables.DnsPort) == 0 {
		return fmt.Errorf("Empty variable dns port")
	} else if len(v.Variables.DnsBinary) == 0 {
		return fmt.Errorf("Empty variable dns binary")
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
	shell *ShellBoss
	cmd   *exec.Cmd
	pid   int
	conf  *SrsServiceConfig
	ctx   ol.Context
	lock  *sync.Mutex
	// the work dir for this process, a sub folder under config.
	workDir string
	// the config for this process, under workDir.
	config string
	// allocated ports(maybe not used).
	ports []int
	pool  *PortPool
	// the used port.
	rtmp  int
	http  int
	api   int
	big   int
	bitch int
	dns   int
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
	return fmt.Sprintf("srs worker, pid=%v, state=%v, version=%v", v.pid, v.state, v.version)
}

func (v *SrsWorker) Close() error {
	v.lock.Lock()
	defer v.lock.Unlock()

	// free the ports.
	for _, p := range v.ports {
		v.pool.Free(p)
	}
	v.ports = nil

	// eventhrouth the process is managed by process pool,
	// we can signal the process to quit when close the worker
	// when process stil alive.
	if v.cmd != nil && v.cmd.ProcessState == nil {
		v.cmd.Process.Kill()
	}
	v.cmd = nil

	// TODO: FIXME: cleanup workdir.

	return nil
}

func (v *SrsWorker) Exec() (err error) {
	if err = v.doExec(); err != nil {
		v.Close()
	}
	return
}

func (v *SrsWorker) doExec() (err error) {
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

	// create work dir and link binaries.
	v.workDir = path.Join(r.WorkDir, fmt.Sprintf("srs/%v", time.Now().Format("2006-01-02-15:04:05.000")))
	if wd := path.Join(v.workDir, "objs/nginx/html"); true {
		if err = os.MkdirAll(wd, 0755); err != nil {
			ol.E(ctx, "create srs dir", wd, "failed, err is", err)
			return
		}
	}
	conf := configTemplate
	conf = strings.Replace(conf, s.Variables.WorkDir, v.workDir, -1)

	// symbol link all binaries to work dir.
	slink := func(workDir, cwdBin string) (err error) {
		if len(cwdBin) == 0 || path.IsAbs(cwdBin) {
			return
		}

		var pwd string
		if pwd, err = os.Getwd(); err != nil {
			ol.E(ctx, "getwd failed, err is", err)
			return
		}
		from, to := path.Join(pwd, cwdBin), path.Join(workDir, cwdBin)

		toDir := path.Dir(to)
		if err = os.MkdirAll(toDir, 0755); err != nil {
			ol.E(ctx, "create srs dir", toDir, "failed, err is", err)
			return
		}

		if err = os.Symlink(from, to); err != nil {
			ol.E(ctx, fmt.Sprintf("symlink %v=%v failed, err is %v", from, to, err))
			return
		}
		return
	}
	slinks := func(workDir string, bins ...string) (err error) {
		for _, bin := range bins {
			if err = slink(workDir, bin); err != nil {
				return
			}
		}
		return
	}
	bins := []string{r.Binary, s.BigBinary, s.BitchBinary, s.FfmpegBinary, s.DnsBinary}
	if err = slinks(v.workDir, bins...); err != nil {
		ol.E(ctx, "symlink failed, err is", err)
		return
	}

	// alloc all ports, althrough maybe not use it.
	if ports, err := v.pool.Alloc(8); err != nil {
		ol.E(ctx, "alloc ports failed, err is", err)
		return err
	} else {
		v.ports = append(v.ports, ports...)
		v.rtmp, v.http, v.api = ports[0], ports[1], ports[2]
		v.big, v.bitch, v.dns = ports[3], ports[4], ports[5]
	}

	// build all port.
	conf = strings.Replace(conf, s.Variables.RtmpPort, strconv.Itoa(v.rtmp), -1)
	conf = strings.Replace(conf, s.Variables.HttpPort, strconv.Itoa(v.http), -1)
	conf = strings.Replace(conf, s.Variables.ApiPort, strconv.Itoa(v.api), -1)
	conf = strings.Replace(conf, s.Variables.BigPort, strconv.Itoa(v.big), -1)
	conf = strings.Replace(conf, s.Variables.BitchPort, strconv.Itoa(v.bitch), -1)
	conf = strings.Replace(conf, s.Variables.DnsPort, strconv.Itoa(v.dns), -1)
	// for http proxy for hls+
	conf = strings.Replace(conf, s.Variables.HttpProxyPort, strconv.Itoa(v.shell.conf.Httplb.Http), -1)
	// for big proxy port.
	conf = strings.Replace(conf, s.Variables.BigProxyPort, strconv.Itoa(v.shell.conf.Apilb.Big), -1)
	// build other variables
	if len(s.BigBinary) > 0 {
		conf = strings.Replace(conf, s.Variables.BigBinary, s.BigBinary, -1)
	}
	if len(s.BitchBinary) > 0 {
		conf = strings.Replace(conf, s.Variables.BitchBinary, s.BitchBinary, -1)
	}
	if len(s.FfmpegBinary) > 0 {
		conf = strings.Replace(conf, s.Variables.FfmpegBinary, s.FfmpegBinary, -1)
	}
	if len(s.DnsBinary) > 0 {
		conf = strings.Replace(conf, s.Variables.DnsBinary, s.DnsBinary, -1)
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

	ports := fmt.Sprintf("rtmp=%v,http=%v,api=%v,big=%v,bitch=%v,dns=%v",
		v.rtmp, v.http, v.api, v.big, v.bitch, v.dns)
	ol.T(ctx, fmt.Sprintf("srs ports(%v), cwd=%v, config=%v", ports, v.workDir, v.config))

	// test the config with srs.
	if b, err := exec.Command(r.Binary, "-t", "-c", v.config).CombinedOutput(); err != nil {
		ol.E(ctx, fmt.Sprintf("test config failed, err is %v, raw log is:\n%v", err, string(b)))
		return err
	}

	// start srs process.
	var cmd *exec.Cmd
	if cmd, err = v.shell.pool.Start(ctx, r.Binary, "-c", v.config); err != nil {
		ol.E(ctx, "exec worker failed, err is", err)
		return
	}
	v.cmd = cmd
	v.pid = cmd.Process.Pid
	ol.T(ctx, fmt.Sprintf("exec worker ok, args=%v, pid=%v", cmd.Args, v.pid))

	return
}
