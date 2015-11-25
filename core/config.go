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
)

// the scope for reload.
const (
	ReloadWorkers = iota
	ReloadLog
	ReloadListen
)

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
	OnReloadGlobal(scope int, cc, pc *Config) error
}

// the reader support c++-style comment,
//      block: /* comments */
//      line: // comments
type Reader struct {
	attention int // attention please, maybe comments.
	f         funcReader
	r         io.Reader
}

func NewReader(r io.Reader) io.Reader {
	return &Reader{r: r}
}

// read from r into p, return actual n bytes, the next handler and err indicates error.
type funcReader func(r io.Reader, p []byte) (n int, next funcReader, err error)

// interface io.Reader
func (v *Reader) Read(p []byte) (n int, err error) {
	var lineReader funcReader
	var blockReader funcReader
	var contentReader funcReader
	var stringReader funcReader

	lineReader = funcReader(func(r io.Reader, p []byte) (n int, next funcReader, err error) {
		b := make([]byte, 1)
		if n, err = io.ReadAtLeast(r, b, 1); err != nil {
			return
		}

		// skip any util \n
		if b[0] != '\n' {
			return 0, lineReader, nil
		}

		return 0, contentReader, nil
	})

	stringReader = funcReader(func(r io.Reader, p []byte) (n int, next funcReader, err error) {
		b := make([]byte, 1)
		if n, err = io.ReadAtLeast(r, b, 1); err != nil {
			return
		}

		p[0] = b[0]

		if b[0] == '"' {
			return 1, contentReader, err
		}

		return 1, stringReader, err
	})

	blockReader = funcReader(func(r io.Reader, p []byte) (n int, next funcReader, err error) {
		if len(p) < v.attention+1 {
			return 0, nil, nil
		}

		// read one byte more.
		b := make([]byte, 1)
		if n, err = io.ReadAtLeast(r, b, 1); err != nil {
			// when EOF, ok for content or content reader,
			// but invalid for block reader.
			if err == io.EOF {
				return 0, nil, errors.New("block comments should not EOF")
			}
			return
		}

		// skip any util */
		if b[0] != '/' && b[0] != '*' {
			return 0, blockReader, nil
		}

		// attention
		if b[0] == '*' {
			v.attention = 1
			return 0, blockReader, nil
		}

		// eof comments.
		if v.attention != 0 && b[0] == '/' {
			v.attention = 0
			return 0, contentReader, nil
		}

		panic(fmt.Sprintf("invalid block, attention=%v, b=%v", v.attention, b[0]))
		return
	})

	contentReader = funcReader(func(r io.Reader, p []byte) (n int, next funcReader, err error) {
		if len(p) < v.attention+1 {
			return 0, nil, nil
		}

		// read one byte more.
		b := make([]byte, 1)
		if n, err = io.ReadAtLeast(r, b, 1); err != nil {
			return
		}

		// 2byte push.
		if v.attention != 0 && b[0] != '/' && b[0] != '*' {
			p[0] = '/'
			p[1] = b[0]
			if b[0] == '"' {
				return 2, stringReader, err
			}
			return 2, contentReader, err
		}

		// 1byte push.
		if v.attention == 0 && b[0] != '/' {
			p[0] = b[0]
			if b[0] == '"' {
				return 1, stringReader, err
			}
			return 1, contentReader, err
		}

		// attention
		if v.attention == 0 && b[0] == '/' {
			v.attention = 1
			return 0, contentReader, err
		}

		// line comments.
		if v.attention != 0 && b[0] == '/' {
			v.attention = 0
			return 0, lineReader, err
		}

		// block comments.
		if v.attention != 0 && b[0] == '*' {
			v.attention = 0
			return 0, blockReader, err
		}

		panic(fmt.Sprintf("invalid content, attention=%v, b=%v", v.attention, b[0]))
		return
	})

	// start using normal byte reader.
	var f funcReader
	if f = v.f; f == nil {
		f = contentReader
	}

	// read util full or no func reader specified.
	for i := 0; f != nil && i < len(p); {
		var ne int
		if ne, f, err = f(v.r, p[i:]); err != nil {
			break
		}

		// apply the consumed bytes.
		n += ne
		i += ne

		// remember the last handler we use.
		if f != nil {
			v.f = f
		}
	}

	return
}

// the vhost section in config.
type Vhost struct {
	Name string `json:"name"`
}

// the config for this application,
// which can load from file in json style,
// and convert to json string.
// @remark user can user the GsConfig object.
type Config struct {
	// the global section.
	Workers int `json:"workers"` // the number of cpus to use

	// the rtmp global section.
	Listen int  `json:"listen"` // the system service RTMP listen port
	Daemon bool `json:"daemon"` // whether enabled the daemon for unix-like os

	// the go section.
	Go struct {
		GcInterval int `json:"gc_interval"` // the gc interval in seconds.
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
	c.Go.GcInterval = 300

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
	for _, v := range c.Vhosts {
		if _, ok := c.vhosts[v.Name]; ok {
			return fmt.Errorf("dup vhost name is", v.Name)
		}

		c.vhosts[v.Name] = v
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

	if c.Go.GcInterval <= 0 || c.Go.GcInterval > 24*3600 {
		return fmt.Errorf("go gc_interval must in (0, 24*3600], actual is %v", c.Go.GcInterval)
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
