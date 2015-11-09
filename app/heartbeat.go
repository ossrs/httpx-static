// The MIT License (MIT)
//
// Copyright (c) 2013-2015 SRS(simple-rtmp-server)
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

package app

import (
	"bytes"
	"encoding/json"
	"github.com/simple-rtmp-server/go-srs/core"
	"net/http"
	"sync"
	"time"
)

type Heartbeat struct {
	ready    bool
	ips      []string
	exportIp string
	lock     sync.Mutex
}

func NewHeartbeat() *Heartbeat {
	return &Heartbeat{
		ips: []string{},
	}
}

func (h *Heartbeat) discoveryCycle(w WorkerContainer) {
	interval := time.Duration(0)
	for {
		select {
		case <-w.QC():
			w.Quit()
			return
		case <-time.After(interval):
			if err := h.discovery(); err != nil {
				core.GsWarn.Println("heartbeat discovery failed, err is", err)
			} else {
				core.GsTrace.Println("local ip is", h.ips)
				interval = 300 * time.Second
			}
		}
	}

	return
}

func (h *Heartbeat) beatCycle(w WorkerContainer) {
	for {
		c := &GsConfig.Heartbeat

		select {
		case <-w.QC():
			w.Quit()
			return
		case <-time.After(time.Millisecond * time.Duration(1000*c.Interval)):
			if !c.Enabled {
				continue
			}

			if err := h.beat(); err != nil {
				core.GsWarn.Println("heartbeat to", c.Url, "every", c.Interval, "seconds failed, err is", err)
			} else {
				core.GsInfo.Println("heartbeat to", c.Url, "every", c.Interval, "seconds")
			}
		}
	}
}

func (h *Heartbeat) discovery() (err error) {
	return
}

func (h *Heartbeat) beat() (err error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if !h.ready || len(h.exportIp) <= 0 {
		core.GsInfo.Println("heartbeat not ready.")
		return
	}

	v := struct {
		DeviceId string `json:"device_id"`
		Ip       string `json:"ip"`
	}{}

	c := &GsConfig.Heartbeat
	v.DeviceId = c.DeviceId
	v.Ip = h.exportIp

	var b []byte
	if b, err = json.Marshal(&v); err != nil {
		return
	}
	core.GsTrace.Println("heartbeat info is", string(b))

	var resp *http.Response
	if resp, err = http.Post(c.Url, core.HttpJson, bytes.NewReader(b)); err != nil {
		return
	}
	defer resp.Body.Close()

	core.GsInfo.Println("heartbeat to", c.Url, "ok")
	return
}
