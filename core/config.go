// The MIT License (MIT)
//
// Copyright (c) 2013-2016 Oryx(ossrs)
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

package core

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	ocore "github.com/ossrs/go-oryx-lib/json"
)

// the scope for reload.
const (
	// global specified.
	ReloadWorkers = iota
	ReloadLog
	ReloadListen
	ReloadCpuProfile
	ReloadGcPercent
	// vhost specified.
	ReloadMwLatency
)

// merged write latency, the group messages to send.
const defaultMwLatency = 350

// the reload handler,
// the client which care about the reload event,
// must implements this interface and then register itself
// to the config.
type ReloadHandler interface {
	// when reload the global scopes,
	// for example, the workers, listen and log.
	// @param scope defined in const ReloadXXX.
	// @param cc the current loaded config, GsConfig.
	// @param pc the previous old config.
	OnReloadGlobal(scope int, cc, pc *Config) (err error)
	// when reload the vhost scopes,
	// for example, the Vhost.Play.MwLatency
	// @param scope defined in const ReloadXXX.
	// @param cc the current loaded config, GsConfig.
	// @param pc the previous old config.
	OnReloadVhost(vhost string, scope int, cc, pc *Config) (err error)
}

// the vhost section in config.
type Vhost struct {
	Name     string `json:"name"`
	Realtime bool   `json:"min_latency"`
	Play     *Play  `json:"play,ommit-empty"`
}

func NewConfVhost() *Vhost {
	return &Vhost{
		Play: NewConfPlay(),
	}
}

type Play struct {
	MwLatency int `json:"mw_latency`
}

func NewConfPlay() *Play {
	return &Play{}
}

// the config for this application,
// which can load from file in json style,
// and convert to json string.
// @remark user can user the GsConfig object.
type Config struct {
	// the global section.
	Workers int `json:"workers"` // the number of cpus to use

	// the rtmp global section.
	Listen    int  `json:"listen"`     // the system service RTMP listen port
	Daemon    bool `json:"daemon"`     // whether enabled the daemon for unix-like os
	ChunkSize int  `json:"chunk_size"` // the output chunk size. [128, 65535].

	// the go section.
	Go struct {
		Writev     bool   `json:"writev"`      // whether use private writev.
		GcTrace    int    `json:"gc_trace"`    // the gc trace interval in seconds.
		GcInterval int    `json:"gc_interval"` // the gc interval in seconds.
		GcPercent  int    `json:"gc_percent"`  // the gc percent.
		CpuProfile string `json:"cpu_profile"` // the cpu profile file.
		MemProfile string `json:"mem_profile"` // the memory profile file.
	}

	// the log config.
	Log struct {
		Tank  string `json:"tank"`  // the log tank, file or console
		Level string `json:"level"` // the log level, info/trace/warn/error
		File  string `json:"file"`  // for log tank file, the log file path.
	} `json:"log"`

	// the heartbeat section.
	Heartbeat struct {
		Enabled  bool    `json:"enabled"`   // whether enable the heartbeat.
		Interval float64 `json:"interval"`  // the heartbeat interval in seconds.
		Url      string  `json:"url"`       // the url to report.
		DeviceId string  `json:"device_id"` // the device id to report.
		Summary  bool    `json:"summaries"` // whether enable the detail summary.
		Listen   int     `json:"listen"`    // the heartbeat http api listen port.
	} `json:"heartbeat"`

	// the stat section.
	Stat struct {
		Network int      `json:"network"` // the network device index to use as exported ip.
		Disks   []string `json:"disk"`    // the disks to stat.
	} `json:"stats"`

	Debug struct {
		RtmpDumpRecv bool `json:"rtmp_dump_recv"`
	} `json:"debug"`

	// the vhosts section.
	Vhosts []*Vhost `json:"vhosts"`

	ctx            Context           `json:"-"`
	conf           string            `json:"-"` // the config file path.
	reloadHandlers []ReloadHandler   `json:"-"`
	vhosts         map[string]*Vhost `json:"-"`
}

// the current global config.
var Conf *Config

func NewConfig(ctx Context) *Config {
	c := &Config{
		ctx:            ctx,
		reloadHandlers: []ReloadHandler{},
		Vhosts:         make([]*Vhost, 0),
		vhosts:         make(map[string]*Vhost),
	}

	return c
}

// get the config file path.
func (v *Config) Conf() string {
	return v.conf
}

func (c *Config) SetDefaults() {
	c.Listen = RtmpListen
	c.Workers = 0
	c.Daemon = true
	c.ChunkSize = 60000
	c.Go.GcInterval = 0

	c.Heartbeat.Enabled = false
	c.Heartbeat.Interval = 9.3
	c.Heartbeat.Url = "http://127.0.0.1:8085/api/v1/servers"
	c.Heartbeat.Summary = false

	c.Stat.Network = 0

	c.Log.Tank = "file"
	c.Log.Level = "trace"
	c.Log.File = "oryx.log"
}

// loads and validate config from config file.
func (v *Config) Loads(conf string) error {
	v.conf = conf

	// set default config values.
	v.SetDefaults()

	// json style should not be *.conf
	if !strings.HasSuffix(conf, ".conf") {
		// read the whole config to []byte.
		if f, err := os.Open(conf); err != nil {
			return err
		} else {
			defer f.Close()

			if err := ocore.Unmarshal(f, v); err != nil {
				return err
			}
		}
	} else {
		// srs-style config.
		var p *srsConfParser
		if f, err := os.Open(conf); err != nil {
			return err
		} else {
			defer f.Close()

			p = NewSrsConfParser(f)
		}

		if err := p.Decode(v); err != nil {
			return err
		}
	}

	// when parse EOF, reparse the config.
	if err := v.reparse(); err != nil {
		return err
	}

	// validate the config.
	if err := v.Validate(); err != nil {
		return err
	}

	return nil
}

// reparse the config, to compatible and better structure.
func (v *Config) reparse() (err error) {
	// check vhost, never dup name.
	for _, p := range v.Vhosts {
		if _, ok := v.vhosts[p.Name]; ok {
			return fmt.Errorf("dup vhost name is", p.Name)
		}

		v.vhosts[p.Name] = p
	}

	// gc percent 0 to use system default(100).
	if v.Go.GcPercent == 0 {
		v.Go.GcPercent = 100
	}

	// default values for vhosts.
	for _, p := range v.Vhosts {
		if p.Play != nil {
			if p.Play.MwLatency == 0 {
				// how many messages send in a group.
				// one message is about 14ms for RTMP audio and video.
				// @remark 0 to disable group messages to send one by one.
				p.Play.MwLatency = defaultMwLatency
			}
		}
	}

	return
}

// validate the config whether ok.
func (v *Config) Validate() error {
	ctx := v.ctx

	if v.Log.Level == "info" {
		Warn.Println(ctx, "info level hurts performance")
	}

	if len(v.Stat.Disks) > 0 {
		Warn.Println(ctx, "stat disks not support")
	}

	if v.Workers < 0 || v.Workers > 64 {
		return fmt.Errorf("workers must in [0, 64], actual is %v", v.Workers)
	}
	if v.Listen <= 0 || v.Listen > 65535 {
		return fmt.Errorf("listen must in (0, 65535], actual is %v", v.Listen)
	}
	if v.ChunkSize < 128 || v.ChunkSize > 65535 {
		return fmt.Errorf("chunk_size must in [128, 65535], actual is %v", v.ChunkSize)
	}

	if v.Go.GcInterval < 0 || v.Go.GcInterval > 24*3600 {
		return fmt.Errorf("go gc_interval must in [0, 24*3600], actual is %v", v.Go.GcInterval)
	}

	if v.Log.Level != "info" && v.Log.Level != "trace" && v.Log.Level != "warn" && v.Log.Level != "error" {
		return fmt.Errorf("log.leve must be info/trace/warn/error, actual is %v", v.Log.Level)
	}
	if v.Log.Tank != "console" && v.Log.Tank != "file" {
		return fmt.Errorf("log.tank must be console/file, actual is %v", v.Log.Tank)
	}
	if v.Log.Tank == "file" && len(v.Log.File) == 0 {
		return errors.New("log.file must not be empty for file tank")
	}

	for i, p := range v.Vhosts {
		if p.Name == "" {
			return fmt.Errorf("the %v vhost is empty", i)
		}
	}

	return nil
}

// whether log tank is file
func (v *Config) LogToFile() bool {
	return v.Log.Tank == "file"
}

// get the log tank writer for specified level.
// the param dw is the default writer.
func (v *Config) LogTank(level string, dw io.Writer) io.Writer {
	if v.Log.Level == "info" {
		return dw
	}
	if v.Log.Level == "trace" {
		if level == "info" {
			return ioutil.Discard
		}
		return dw
	}
	if v.Log.Level == "warn" {
		if level == "info" || level == "trace" {
			return ioutil.Discard
		}
		return dw
	}
	if v.Log.Level == "error" {
		if level != "error" {
			return ioutil.Discard
		}
		return dw
	}

	return ioutil.Discard
}

// subscribe the reload event,
// when got reload event, notify all handlers.
func (v *Config) Subscribe(h ReloadHandler) {
	// ignore exists.
	for _, v := range v.reloadHandlers {
		if v == h {
			return
		}
	}

	v.reloadHandlers = append(v.reloadHandlers, h)
}

func (v *Config) Unsubscribe(h ReloadHandler) {
	for i, p := range v.reloadHandlers {
		if p == h {
			v.reloadHandlers = append(v.reloadHandlers[:i], v.reloadHandlers[i+1:]...)
			return
		}
	}
}

func (v *Config) Reload(cc *Config) (err error) {
	pc := v
	ctx := v.ctx

	if cc.Workers != pc.Workers {
		for _, h := range cc.reloadHandlers {
			if err = h.OnReloadGlobal(ReloadWorkers, cc, pc); err != nil {
				return
			}
		}
		Trace.Println(ctx, "reload apply workers ok")
	} else {
		Info.Println(ctx, "reload ignore workers")
	}

	if cc.Log.File != pc.Log.File || cc.Log.Level != pc.Log.Level || cc.Log.Tank != pc.Log.Tank {
		for _, h := range cc.reloadHandlers {
			if err = h.OnReloadGlobal(ReloadLog, cc, pc); err != nil {
				return
			}
		}
		Trace.Println(ctx, "reload apply log ok")
	} else {
		Info.Println(ctx, "reload ignore log")
	}

	if cc.Listen != pc.Listen {
		for _, h := range cc.reloadHandlers {
			if err = h.OnReloadGlobal(ReloadListen, cc, pc); err != nil {
				return
			}
		}
		Trace.Println(ctx, "reload apply listen ok")
	} else {
		Info.Println(ctx, "reload ignore listen")
	}

	if cc.Go.CpuProfile != pc.Go.CpuProfile {
		for _, h := range cc.reloadHandlers {
			if err = h.OnReloadGlobal(ReloadCpuProfile, cc, pc); err != nil {
				return
			}
		}
		Trace.Println(ctx, "reload apply cpu profile ok")
	} else {
		Info.Println(ctx, "reload ignore cpu profile")
	}

	if cc.Go.GcPercent != pc.Go.GcPercent {
		for _, h := range cc.reloadHandlers {
			if err = h.OnReloadGlobal(ReloadGcPercent, cc, pc); err != nil {
				return
			}
		}
		Trace.Println(ctx, "reload apply gc percent ok")
	} else {
		Info.Println(ctx, "reload ignore gc percent")
	}

	// vhost specified.
	for k, cv := range cc.vhosts {
		if pv := pc.vhosts[k]; cv.Play != nil && pv.Play != nil && cv.Play.MwLatency != pv.Play.MwLatency {
			for _, h := range cc.reloadHandlers {
				if err = h.OnReloadVhost(k, ReloadMwLatency, cc, pc); err != nil {
					return
				}
			}
			Trace.Println(ctx, "reload apply vhost.play.mw-latency ok")
		} else {
			Info.Println(ctx, "reload ignore vhost.play.mw-latency")
		}
	}

	return
}

func (v *Config) Vhost(name string) (*Vhost, error) {
	if v, ok := v.vhosts[name]; ok {
		return v, nil
	}

	if name != RtmpDefaultVhost {
		return v.Vhost(RtmpDefaultVhost)
	}

	return nil, VhostNotFoundError
}

func (v *Config) VhostGroupMessages(vhost string) (n int, err error) {
	var p *Vhost
	if p, err = v.Vhost(vhost); err != nil {
		return
	}

	if p.Play == nil {
		return defaultMwLatency / 14, nil
	}
	return p.Play.MwLatency / 14, nil
}

func (v *Config) VhostRealtime(vhost string) (r bool, err error) {
	var p *Vhost
	if p, err = v.Vhost(vhost); err != nil {
		return
	}

	return p.Realtime, nil
}
