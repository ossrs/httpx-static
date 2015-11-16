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

package app

import (
	"bytes"
	"encoding/json"
	"github.com/ossrs/go-oryx/core"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"
)

type Heartbeat struct {
	ips      []string
	exportIp string
	lock     sync.Mutex
}

func NewHeartbeat() *Heartbeat {
	return &Heartbeat{
		ips: []string{},
	}
}

func (h *Heartbeat) discoveryCycle(w core.WorkerContainer) {
	interval := time.Duration(0)
	for {
		select {
		case <-w.QC():
			w.Quit()
			return
		case <-time.After(interval):
			core.Info.Println("start to discovery network every", interval)

			if err := h.discovery(); err != nil {
				core.Warn.Println("heartbeat discovery failed, err is", err)
			} else {
				if len(h.ips) <= 0 {
					interval = 3 * time.Second
					continue
				}
				core.Trace.Println("local ip is", h.ips, "exported", h.exportIp)
				interval = 300 * time.Second
			}
		}
	}

	return
}

func (h *Heartbeat) beatCycle(w core.WorkerContainer) {
	for {
		c := &core.Conf.Heartbeat

		select {
		case <-w.QC():
			w.Quit()
			return
		case <-time.After(time.Millisecond * time.Duration(1000*c.Interval)):
			if !c.Enabled {
				continue
			}

			core.Info.Println("start to heartbeat every", c.Interval)

			if err := h.beat(); err != nil {
				core.Warn.Println("heartbeat to", c.Url, "every", c.Interval, "failed, err is", err)
			} else {
				core.Info.Println("heartbeat to", c.Url, "every", c.Interval)
			}
		}
	}
}

func (h *Heartbeat) discovery() (err error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	// check whether the ip is ok to export.
	vf := func(ip net.IP) bool {
		return ip != nil && ip.To4() != nil && !ip.IsLoopback()
	}

	// fetch the ip from addr interface.
	ipf := func(addr net.Addr) (string, bool) {
		if v, ok := addr.(*net.IPNet); ok && vf(v.IP) {
			return v.IP.String(), true
		} else if v, ok := addr.(*net.IPAddr); ok && vf(v.IP) {
			return v.IP.String(), true
		} else {
			return "", false
		}
	}

	var ifaces []net.Interface
	if ifaces, err = net.Interfaces(); err != nil {
		return
	}

	h.ips = []string{}
	for _, iface := range ifaces {
		var addrs []net.Addr
		if addrs, err = iface.Addrs(); err != nil {
			return
		}

		// dumps all network interfaces.
		for _, addr := range addrs {
			if v, ok := ipf(addr); ok {
				core.Trace.Println("iface", iface.Name, "ip is", v)
				h.ips = append(h.ips, v)
			} else {
				core.Info.Println("iface", iface.Name, addr, reflect.TypeOf(addr))
			}
		}
	}

	// choose one as exported network address.
	if len(h.ips) > 0 {
		h.exportIp = h.ips[core.Conf.Stat.Network%len(h.ips)]
	}
	return
}

func (h *Heartbeat) beat() (err error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if len(h.exportIp) <= 0 {
		core.Info.Println("heartbeat not ready.")
		return
	}

	v := struct {
		DeviceId string      `json:"device_id"`
		Ip       string      `json:"ip"`
		Summary  interface{} `json:"summaries,omitempty"`
	}{}

	c := &core.Conf.Heartbeat
	v.DeviceId = c.DeviceId
	v.Ip = h.exportIp

	if c.Summary {
		s := NewSummary()
		s.Ok = true

		v.Summary = struct {
			Code int      `json:"code"`
			Data *Summary `json:"data"`
		}{
			Code: 0,
			Data: s,
		}
	}

	var b []byte
	if b, err = json.Marshal(&v); err != nil {
		return
	}
	core.Info.Println("heartbeat info is", string(b))

	var resp *http.Response
	if resp, err = http.Post(c.Url, core.HttpJson, bytes.NewReader(b)); err != nil {
		return
	}
	defer resp.Body.Close()

	core.Info.Println("heartbeat to", c.Url, "ok")
	return
}
