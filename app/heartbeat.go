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
	ctx      core.Context
	ips      []string
	exportIp string
	lock     sync.Mutex
}

func NewHeartbeat(ctx core.Context) *Heartbeat {
	return &Heartbeat{
		ctx: ctx,
		ips: []string{},
	}
}

const (
	discoveryEmptyInterval   = 3 * time.Second
	discoveryRefreshInterval = 3600 * time.Second
)

func (v *Heartbeat) discoveryCycle(w core.WorkerContainer) {
	ctx := v.ctx

	interval := time.Duration(0)
	for {
		select {
		case <-w.QC():
			w.Quit()
			return
		case <-time.After(interval):
			core.Info.Println(ctx, "start to discovery network every", interval)

			if err := v.discovery(); err != nil {
				core.Warn.Println(ctx, "heartbeat discovery failed, err is", err)
			} else {
				if len(v.ips) <= 0 {
					interval = discoveryEmptyInterval
					continue
				}
				core.Trace.Println(ctx, "local ip is", v.ips, "exported", v.exportIp)
				interval = discoveryRefreshInterval
			}
		}
	}

	return
}

func (v *Heartbeat) beatCycle(w core.WorkerContainer) {
	ctx := v.ctx

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

			core.Info.Println(ctx, "start to heartbeat every", c.Interval)

			if err := v.beat(); err != nil {
				core.Warn.Println(ctx, "heartbeat to", c.Url, "every", c.Interval, "failed, err is", err)
			} else {
				core.Info.Println(ctx, "heartbeat to", c.Url, "every", c.Interval)
			}
		}
	}
}

func (v *Heartbeat) discovery() (err error) {
	ctx := v.ctx

	v.lock.Lock()
	defer v.lock.Unlock()

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

	v.ips = []string{}
	for _, iface := range ifaces {
		var addrs []net.Addr
		if addrs, err = iface.Addrs(); err != nil {
			return
		}

		// dumps all network interfaces.
		for _, addr := range addrs {
			if p, ok := ipf(addr); ok {
				core.Trace.Println(ctx, "iface", iface.Name, "ip is", p)
				v.ips = append(v.ips, p)
			} else {
				core.Info.Println(ctx, "iface", iface.Name, addr, reflect.TypeOf(addr))
			}
		}
	}

	// choose one as exported network address.
	if len(v.ips) > 0 {
		v.exportIp = v.ips[core.Conf.Stat.Network%len(v.ips)]
	}
	return
}

func (v *Heartbeat) beat() (err error) {
	ctx := v.ctx

	v.lock.Lock()
	defer v.lock.Unlock()

	if len(v.exportIp) <= 0 {
		core.Info.Println(ctx, "heartbeat not ready.")
		return
	}

	p := struct {
		DeviceId string      `json:"device_id"`
		Ip       string      `json:"ip"`
		Summary  interface{} `json:"summaries,omitempty"`
	}{}

	c := &core.Conf.Heartbeat
	p.DeviceId = c.DeviceId
	p.Ip = v.exportIp

	if c.Summary {
		s := NewSummary()
		s.Ok = true

		p.Summary = struct {
			Code int      `json:"code"`
			Data *Summary `json:"data"`
		}{
			Code: 0,
			Data: s,
		}
	}

	var b []byte
	if b, err = json.Marshal(&p); err != nil {
		return
	}
	core.Info.Println(ctx, "heartbeat info is", string(b))

	var resp *http.Response
	if resp, err = http.Post(c.Url, core.HttpJson, bytes.NewReader(b)); err != nil {
		return
	}
	defer resp.Body.Close()

	core.Info.Println(ctx, "heartbeat to", c.Url, "ok")
	return
}
