// The MIT License (MIT)
//
// Copyright (c) 2013-2015 Oryx(ossrs)
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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"bufio"
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

// the state for json with comment parser.
type jsonState uint8
const (
	JsInit jsonState = iota
	JsText
	JsNoComment
	JsBlockComment
	JsLineComment
)

// the reader support c++-style comment,
//      block: /* comments */
//      line: // comments
type jsonCommentReader struct {
	st jsonState
	br *bufio.Reader
}

func NewReader(r io.Reader) io.Reader {
	return &jsonCommentReader{
		br: bufio.NewReader(r),
		st: JsInit,
	}
}

// interface io.Reader
func (v *jsonCommentReader) Read(p []byte) (n int, err error) {
	startsWith := func(r *bufio.Reader, flags ...byte) (match bool, err error) {
		var pk []byte
		if pk,err = r.Peek(len(flags)); err != nil {
			return
		}
		for i := 0; i < len(pk); i++ {
			if pk[i] != flags[i] {
				return false,nil
			}
		}
		return true,nil
	}
	discardUtil := func(r *bufio.Reader, flags ...byte) (err error) {
		for {
			var match bool
			if match,err = startsWith(r, flags...); err != nil {
				return
			} else if match {
				return nil
			}
			if _,err = r.Discard(1); err != nil {
				return
			}
		}
		return
	}
	discardUtilAny := func(r *bufio.Reader, flags ...byte) (err error) {
		var pk []byte
		for {
			if pk,err = r.Peek(1); err != nil {
				return
			}
			for _,v := range flags {
				if pk[0] == v {
					return
				}
			}
			if _,err = r.Discard(1); err != nil {
				return
			}
		}
		return
	}
	discardUtilNot := func(r *bufio.Reader, flags ...byte) (err error) {
		var pk []byte
		for {
			if pk,err = r.Peek(1); err != nil {
				return
			}
			var match bool
			for _,v := range flags {
				if pk[0] == v {
					match = true
					break
				}
			}
			if !match {
				return
			}
			if _,err = r.Discard(1); err != nil {
				return
			}
		}
		return
	}

	for n < len(p) {
		// from init to working state.
		if v.st == JsInit {
			var match bool
			if match,err = startsWith(v.br, '/', '*'); err != nil {
				if err == io.EOF {
					v.st = JsText
					continue
				}
				return
			} else if match {
				v.st = JsBlockComment
			} else if match,err = startsWith(v.br, '/', '/'); err != nil {
				if err == io.EOF {
					v.st = JsText
					continue
				}
				return
			} else if match {
				v.st = JsLineComment
			} else {
				v.st = JsText
				continue
			}
			if _, err = v.br.Discard(2); err != nil {
				return
			}
		}

		// block comment state, expect eof with */
		if v.st == JsBlockComment {
			if err = discardUtil(v.br, '*', '/'); err != nil {
				return
			}
			if _,err = v.br.Discard(2); err != nil {
				return
			}
		}

		// discard all newline, like \n \r
		if v.st == JsLineComment {
			if err = discardUtilAny(v.br, '\n', '\r'); err != nil {
				return
			}
			if err = discardUtilNot(v.br, '\n', '\r'); err != nil {
				return
			}
		}

		// append text.
		if v.st == JsText || v.st == JsNoComment {
			var ch byte
			if ch,err = v.br.ReadByte(); err != nil {
				return
			}
			if ch == '"' {
				if v.st == JsText {
					v.st = JsNoComment
				} else {
					v.st = JsText
				}
			}
			p[n] = ch
			n++
		}

		// reset to init state.
		if v.st != JsNoComment {
			v.st = JsInit
		}
	}
	return
}

// the vhost section in config.
type Vhost struct {
	Name string `json:"name"`
	Play *Play  `json:"play"`
}

type Play struct {
	MwLatency int `json:"mw_latency`
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
	} `json:"heartbeat"`

	// the stat section.
	Stat struct {
		Network int      `json:"network"` // the network device index to use as exported ip.
		Disks   []string `json:"disk"`    // the disks to stat.
	} `json:"stats"`

	// the vhosts section.
	Vhosts []*Vhost `json:"vhosts"`

	conf           string            `json:"-"` // the config file path.
	reloadHandlers []ReloadHandler   `json:"-"`
	vhosts         map[string]*Vhost `json:"-"`
}

// the current global config.
var Conf = NewConfig()

func NewConfig() *Config {
	c := &Config{
		reloadHandlers: []ReloadHandler{},
		Vhosts:         make([]*Vhost, 0),
		vhosts:         make(map[string]*Vhost),
	}

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

	return c
}

// get the config file path.
func (c *Config) Conf() string {
	return c.conf
}

// loads and validate config from config file.
func (c *Config) Loads(conf string) error {
	c.conf = conf

	// read the whole config to []byte.
	var d *json.Decoder
	if f, err := os.Open(conf); err != nil {
		return err
	} else {
		defer f.Close()

		d = json.NewDecoder(NewReader(f))
		//d = json.NewDecoder(f)
	}

	// decode config from stream.
	if err := d.Decode(c); err != nil {
		return err
	}

	// when parse EOF, reparse the config.
	if err := c.reparse(); err != nil {
		return err
	}

	// validate the config.
	if err := c.Validate(); err != nil {
		return err
	}

	return nil
}

// reparse the config, to compatible and better structure.
func (c *Config) reparse() (err error) {
	// check vhost, never dup name.
	for _, v := range c.Vhosts {
		if _, ok := c.vhosts[v.Name]; ok {
			return fmt.Errorf("dup vhost name is", v.Name)
		}

		c.vhosts[v.Name] = v
	}

	// gc percent 0 to use system default(100).
	if c.Go.GcPercent == 0 {
		c.Go.GcPercent = 100
	}

	// default values for vhosts.
	for _, v := range c.Vhosts {
		if v.Play != nil {
			if v.Play.MwLatency == 0 {
				// how many messages send in a group.
				// one message is about 14ms for RTMP audio and video.
				// @remark 0 to disable group messages to send one by one.
				v.Play.MwLatency = defaultMwLatency
			}
		}
	}

	return
}

// validate the config whether ok.
func (c *Config) Validate() error {
	if c.Log.Level == "info" {
		Warn.Println("info level hurts performance")
	}

	if len(c.Stat.Disks) > 0 {
		Warn.Println("stat disks not support")
	}

	if c.Workers < 0 || c.Workers > 64 {
		return fmt.Errorf("workers must in [0, 64], actual is %v", c.Workers)
	}
	if c.Listen <= 0 || c.Listen > 65535 {
		return fmt.Errorf("listen must in (0, 65535], actual is %v", c.Listen)
	}
	if c.ChunkSize < 128 || c.ChunkSize > 65535 {
		return fmt.Errorf("chunk_size must in [128, 65535], actual is %v", c.ChunkSize)
	}

	if c.Go.GcInterval < 0 || c.Go.GcInterval > 24*3600 {
		return fmt.Errorf("go gc_interval must in [0, 24*3600], actual is %v", c.Go.GcInterval)
	}

	if c.Log.Level != "info" && c.Log.Level != "trace" && c.Log.Level != "warn" && c.Log.Level != "error" {
		return fmt.Errorf("log.leve must be info/trace/warn/error, actual is %v", c.Log.Level)
	}
	if c.Log.Tank != "console" && c.Log.Tank != "file" {
		return fmt.Errorf("log.tank must be console/file, actual is %v", c.Log.Tank)
	}
	if c.Log.Tank == "file" && len(c.Log.File) == 0 {
		return errors.New("log.file must not be empty for file tank")
	}

	for i, v := range c.Vhosts {
		if v.Name == "" {
			return fmt.Errorf("the %v vhost is empty", i)
		}
	}

	return nil
}

// whether log tank is file
func (c *Config) LogToFile() bool {
	return c.Log.Tank == "file"
}

// get the log tank writer for specified level.
// the param dw is the default writer.
func (c *Config) LogTank(level string, dw io.Writer) io.Writer {
	if c.Log.Level == "info" {
		return dw
	}
	if c.Log.Level == "trace" {
		if level == "info" {
			return ioutil.Discard
		}
		return dw
	}
	if c.Log.Level == "warn" {
		if level == "info" || level == "trace" {
			return ioutil.Discard
		}
		return dw
	}
	if c.Log.Level == "error" {
		if level != "error" {
			return ioutil.Discard
		}
		return dw
	}

	return ioutil.Discard
}

// subscribe the reload event,
// when got reload event, notify all handlers.
func (c *Config) Subscribe(h ReloadHandler) {
	// ignore exists.
	for _, v := range c.reloadHandlers {
		if v == h {
			return
		}
	}

	c.reloadHandlers = append(c.reloadHandlers, h)
}

func (c *Config) Unsubscribe(h ReloadHandler) {
	for i, v := range c.reloadHandlers {
		if v == h {
			c.reloadHandlers = append(c.reloadHandlers[:i], c.reloadHandlers[i+1:]...)
			return
		}
	}
}

func (pc *Config) Reload(cc *Config) (err error) {
	if cc.Workers != pc.Workers {
		for _, h := range cc.reloadHandlers {
			if err = h.OnReloadGlobal(ReloadWorkers, cc, pc); err != nil {
				return
			}
		}
		Trace.Println("reload apply workers ok")
	} else {
		Info.Println("reload ignore workers")
	}

	if cc.Log.File != pc.Log.File || cc.Log.Level != pc.Log.Level || cc.Log.Tank != pc.Log.Tank {
		for _, h := range cc.reloadHandlers {
			if err = h.OnReloadGlobal(ReloadLog, cc, pc); err != nil {
				return
			}
		}
		Trace.Println("reload apply log ok")
	} else {
		Info.Println("reload ignore log")
	}

	if cc.Listen != pc.Listen {
		for _, h := range cc.reloadHandlers {
			if err = h.OnReloadGlobal(ReloadListen, cc, pc); err != nil {
				return
			}
		}
		Trace.Println("reload apply listen ok")
	} else {
		Info.Println("reload ignore listen")
	}

	if cc.Go.CpuProfile != pc.Go.CpuProfile {
		for _, h := range cc.reloadHandlers {
			if err = h.OnReloadGlobal(ReloadCpuProfile, cc, pc); err != nil {
				return
			}
		}
		Trace.Println("reload apply cpu profile ok")
	} else {
		Info.Println("reload ignore cpu profile")
	}

	if cc.Go.GcPercent != pc.Go.GcPercent {
		for _, h := range cc.reloadHandlers {
			if err = h.OnReloadGlobal(ReloadGcPercent, cc, pc); err != nil {
				return
			}
		}
		Trace.Println("reload apply gc percent ok")
	} else {
		Info.Println("reload ignore gc percent")
	}

	// vhost specified.
	for k, cv := range cc.vhosts {
		if pv := pc.vhosts[k]; cv.Play != nil && pv.Play != nil && cv.Play.MwLatency != pv.Play.MwLatency {
			for _, h := range cc.reloadHandlers {
				if err = h.OnReloadVhost(k, ReloadMwLatency, cc, pc); err != nil {
					return
				}
			}
			Trace.Println("reload apply vhost.play.mw-latency ok")
		} else {
			Info.Println("reload ignore vhost.play.mw-latency")
		}
	}

	return
}

func (c *Config) Vhost(name string) (*Vhost, error) {
	if v, ok := c.vhosts[name]; ok {
		return v, nil
	}

	if name != RtmpDefaultVhost {
		return c.Vhost(RtmpDefaultVhost)
	}

	return nil, VhostNotFoundError
}

func (c *Config) VhostGroupMessages(vhost string) (n int, err error) {
	var v *Vhost
	if v, err = c.Vhost(vhost); err != nil {
		return
	}

	if v.Play == nil {
		return defaultMwLatency / 14, nil
	}
	return v.Play.MwLatency / 14, nil
}
