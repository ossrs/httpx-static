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
	"io"
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"errors"
)

type srsConfDirective struct {
	name string
	args []string
	directives []*srsConfDirective
}

func NewSrsConfDirective() *srsConfDirective {
	return &srsConfDirective{
		args: make([]string, 0),
		directives: make([]*srsConfDirective, 0),
	}
}

func (v *srsConfDirective) Arg0() string {
	if len(v.args) > 0 {
		return v.args[0]
	}
	return ""
}

func (v *srsConfDirective) Arg1() string {
	if len(v.args) > 1 {
		return v.args[1]
	}
	return ""
}

func (v *srsConfDirective) Arg2() string {
	if len(v.args) > 2 {
		return v.args[2]
	}
	return ""
}

func (v *srsConfDirective) Get(name string, args ...string) *srsConfDirective {
	mainLoop:
	for _,v := range v.directives {
		if v.name != name {
			continue
		}
		if len(args) > len(v.args) {
			continue
		}
		for i,arg := range args {
			if arg != v.args[i] {
				continue mainLoop
			}
		}
		return v
	}
	return nil
}

var stringNotMatch = errors.New("string not match")
var scanString = func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if bytes.IndexAny(data, "'\"") != 0 {
		if atEOF {
			return 0, nil, stringNotMatch
		}
		return 0,nil,nil
	}

	if i := bytes.IndexByte(data[1:], data[0]); i < 0 {
		if atEOF {
			return 0, nil, stringNotMatch
		}
		return 0,nil,nil
	} else {
		return i+2,data[:i+2], nil
	}
	return
}

var srsConfEndOfObject = errors.New("object EOF")
var srsConfStartOfObject = errors.New("object START")
var srsConfEndOfDirective = errors.New("directive EOF")
var srsConfInvalid = errors.New("invalid srs config")
func (v *srsConfDirective) Parse(s *bufio.Scanner) (err error) {
	// name and args.
	s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error){
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if bytes.IndexAny(data, "'\"") == 0 {
			return scanString(data, atEOF)
		}

		advance,token,err = bufio.ScanWords(data, atEOF)
		if err != nil || token == nil {
			return
		}

		i := bytes.IndexByte(token, '{')
		if i == -1 {
			i = bytes.IndexByte(token, ';')
		}

		if i == 0 {
			return 1,token[:1],nil
		} else if i > 0 {
			token = token[:i]
			advance = bytes.Index(data, token) + len(token)
			return advance, token, nil
		}
		return
	})

	for s.Scan() {
		str := s.Text()
		str = strings.TrimSpace(str)
		str = strings.Trim(str, "'\"")

		if str == "" {
			continue
		} else if str == "{" {
			err = srsConfStartOfObject
			break
		} else if str == "}" {
			return srsConfEndOfObject
		} else if str == ";" {
			err = srsConfEndOfDirective
			break
		}

		if len(v.name) == 0 {
			v.name = str
		} else {
			v.args = append(v.args, str)
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	// for we got directive end.
	if err == srsConfEndOfDirective {
		return nil
	}

	// for sub directives.
	if err == srsConfStartOfObject {
		for {
			dir := NewSrsConfDirective()
			if err := dir.Parse(s); err != nil {
				if err == srsConfEndOfObject {
					return nil
				}
				return err
			}
			if len(dir.name) == 0 {
				return srsConfInvalid
			}
			v.directives = append(v.directives, dir)
		}
	}

	// noting started, EOF.
	if len(v.name) == 0 && len(v.args) == 0 && len(v.directives) == 0 {
		return io.EOF
	}

	return srsConfInvalid
}

// the reader support bash-style comment,
//      line: # comments
func NewSrsConfCommentReader(r io.Reader) io.Reader {
	startMatches := [][]byte{ []byte("'"), []byte("\""), []byte("#"), }
	endMatches := [][]byte{ []byte("'"), []byte("\""), []byte("\n"), }
	isComments := []bool { false, false, true, }
	requiredMatches := []bool { true, true, false, }
	return NewCommendReader(r, startMatches, endMatches, isComments, requiredMatches)
}

// parse the srs style config.
type SrsConfParser struct {
	r io.Reader
}

func NewSrsConfParser(r io.Reader) *SrsConfParser {
	return &SrsConfParser{
		r: NewSrsConfCommentReader(r),
	}
}

func (v *SrsConfParser) Decode(c *Config) (err error) {
	root := NewSrsConfDirective()

	s := bufio.NewScanner(v.r)
	for {
		dir := NewSrsConfDirective()
		if err := dir.Parse(s); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		root.directives = append(root.directives, dir)
	}

	if d := root.Get("listen"); d != nil {
		if c.Listen,err = strconv.Atoi(d.Arg0()); err != nil {
			return
		}
	}

	if d := root.Get("srs_log_tank"); d != nil {
		c.Log.Tank = d.Arg0()
	}
	if d := root.Get("srs_log_level"); d != nil {
		c.Log.Level = d.Arg0()
	}
	if d := root.Get("srs_log_file"); d != nil {
		c.Log.File = d.Arg0()
	}

	if d := root.Get("chunk_size"); d != nil {
		if c.ChunkSize,err = strconv.Atoi(d.Arg0()); err != nil {
			return
		}
	}
	if d := root.Get("daemon"); d != nil {
		c.Daemon = srs_switch2bool(d.Arg0())
	}

	if d := root.Get("heartbeat"); d != nil {
		if d := d.Get("enabled"); d != nil {
			c.Heartbeat.Enabled = srs_switch2bool(d.Arg0())
		}
		if d := d.Get("interval"); d != nil {
			if c.Heartbeat.Interval,err = strconv.ParseFloat(d.Arg0(), 64); err != nil {
				return
			}
		}
		if d := d.Get("device_id"); d != nil {
			c.Heartbeat.DeviceId = d.Arg0()
		}
		if d := d.Get("url"); d != nil {
			c.Heartbeat.Url = d.Arg0()
		}
		if d := d.Get("summaries"); d != nil {
			c.Heartbeat.Summary = srs_switch2bool(d.Arg0())
		}
	}

	if d := root.Get("stats"); d != nil {
		if d := d.Get("network"); d != nil {
			if c.Stat.Network,err = strconv.Atoi(d.Arg0()); err != nil {
				return
			}
		}
		if d := d.Get("disk"); d != nil {
			c.Stat.Disks = make([]string, 0)
			for _,v := range d.args {
				c.Stat.Disks = append(c.Stat.Disks, v)
			}
		}
	}

	c.Vhosts = make([]*Vhost, 0)
	for _, d := range root.directives {
		if d.name != "vhost" {
			continue
		}

		vhost := NewConfVhost()
		c.Vhosts = append(c.Vhosts, vhost)
		vhost.Name = d.Arg0()

		if d := d.Get("play"); d != nil {
			if d := d.Get("mw_latency"); d != nil {
				if vhost.Play.MwLatency,err = strconv.Atoi(d.Arg0()); err != nil {
					return
				}
			}
		}
	}

	return
}

func srs_switch2bool(v string) bool {
	return v == "on"
}

