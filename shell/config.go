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
 The config object for shell.
*/
package main

import (
	"encoding/json"
	"fmt"
	oj "github.com/ossrs/go-oryx-lib/json"
	ol "github.com/ossrs/go-oryx-lib/logger"
	"github.com/ossrs/go-oryx/kernel"
	"os"
	"os/exec"
)

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
		Api     int    `json:"api"`
		Rtmp    int    `json:"rtmp"`
	} `json:"rtmplb"`
	Httplb struct {
		Enabled bool   `json:"enabled"`
		Binary  string `json:"binary"`
		Config  string `json:"config"`
		Api     int    `json:"api"`
		Http    int    `json:"http"`
	} `json:"httplb"`
	Apilb struct {
		Enabled bool   `json:"enabled"`
		Binary  string `json:"binary"`
		Config  string `json:"config"`
		Api     int    `json:"api"`
		Srs     int    `json:"srs"`
		Big     int    `json:"big"`
	} `json:"apilb"`
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
	Api string `json:"api"`
}

func (v *ShellConfig) String() string {
	var rtmplb, httplb, apilb, worker string
	if r := &v.Rtmplb; true {
		rtmplb = fmt.Sprintf("rtmplb(%v,binary=%v,config=%v,api=%v,rtmp=%v)",
			r.Enabled, r.Binary, r.Config, r.Api, r.Rtmp)
	}
	if r := &v.Httplb; true {
		httplb = fmt.Sprintf("httplb(%v,binary=%v,config=%v,api=%v,http=%v)",
			r.Enabled, r.Binary, r.Config, r.Api, r.Http)
	}
	if r := &v.Apilb; true {
		apilb = fmt.Sprintf("apilb(%v,binary=%v,config=%v,api=%v,srs=%v,big=%v)",
			r.Enabled, r.Binary, r.Config, r.Api, r.Srs, r.Big)
	}
	if r := &v.Worker; true {
		worker = fmt.Sprintf("worker(%v,provider=%v,binary=%v,config=%v,dir=%v,ports=[%v,%v],service=%v)",
			r.Enabled, r.Provider, r.Binary, r.Config, r.WorkDir, r.Ports.Start, r.Ports.Stop, r.Service)
	}
	return fmt.Sprintf("%v, api=%v, %v, %v, %v, %v", &v.Config, v.Api, rtmplb, httplb, apilb, worker)
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
			v.Worker.Service = NewSrsServiceConfig()
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
		if r.Api == 0 {
			return fmt.Errorf("Empty rtmplb api port")
		}
		if r.Rtmp == 0 {
			return fmt.Errorf("Empty rtmplb rtmp port")
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
		if r.Api == 0 {
			return fmt.Errorf("Empty httplb api port")
		}
		if r.Http == 0 {
			return fmt.Errorf("Empty httplb http port")
		}
	}

	if r := &v.Apilb; r.Enabled {
		if len(r.Binary) == 0 {
			return fmt.Errorf("Empty apilb binary")
		}
		if len(r.Config) == 0 {
			return fmt.Errorf("Empty apilb config")
		}
		if r.Api == 0 {
			return fmt.Errorf("Empty apilb api")
		}
		if r.Srs == 0 {
			return fmt.Errorf("Empty srs api")
		}
		if r.Big == 0 {
			return fmt.Errorf("Empty big api")
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

	if len(v.Api) == 0 {
		return fmt.Errorf("Empty api listen")
	}

	return
}
