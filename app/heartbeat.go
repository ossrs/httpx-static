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

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	ocore "github.com/ossrs/go-oryx-lib/http"
	"github.com/ossrs/go-oryx/core"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"
)

// IfaceType interface type
type IfaceType uint8

const (
	// IfaceInternet Internet/WAN interface
	IfaceInternet IfaceType = iota
	// IfaceIntranet Intranet/LAN interface
	IfaceIntranet
	// IfaceUnknown Unknown interface
	IfaceUnknown
)

func (v IfaceType) String() string {
	switch v {
	case IfaceInternet:
		return "Internet"
	case IfaceIntranet:
		return "Intranet"
	default:
		return "Unknown"
	}
}

// NetworkIface Network interface
type NetworkIface struct {
	// interface name.
	Ifname string
	// the ip address of interface.
	IP string
	// the mac address of interface.
	Mac string
	// whether the interface ip is public.
	Internet IfaceType
}

func (v *NetworkIface) String() string {
	return fmt.Sprintf("%v/%v/%v/%v", v.Ifname, v.IP, v.Mac, v.Internet)
}

// Heartbeat establishes the heartbeat interface
type Heartbeat struct {
	ctx      core.Context
	ips      []*NetworkIface
	devices  map[string]interface{}
	exportIP *NetworkIface
	lock     sync.Mutex
}

// NewHeartbeat returns a new Heartbeat
func NewHeartbeat(ctx core.Context) *Heartbeat {
	return &Heartbeat{
		ctx:      ctx,
		ips:      make([]*NetworkIface, 0),
		devices:  make(map[string]interface{}),
		exportIP: nil,
	}
}

// Initialize a heartbeat monitor
func (v *Heartbeat) Initialize(w core.WorkerContainer) (err error) {
	ctx := v.ctx
	c := &core.Conf.Heartbeat

	if !c.Enabled {
		return
	}
	if c.Listen <= 0 {
		return
	}

	var l net.Listener
	ep := fmt.Sprintf(":%v", c.Listen)
	if l, err = net.Listen("tcp", ep); err != nil {
		core.Error.Println(ctx, "htbt listen at", ep, "failed. err is", err)
		return
	}
	core.Trace.Println(ctx, "htbt(api) listen at", fmt.Sprintf("tcp://%v", c.Listen))

	isListenerClosed := false
	w.GFork("htbt(api)", func(w core.WorkerContainer) {
		var err error

		h := http.NewServeMux()
		h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			p := struct {
				Urls map[string]string `json:"urls"`
			}{
				Urls: map[string]string{
					"/api/v1/htbt/devices": "each device is object(id:string,data:object).",
				},
			}

			ocore.Data(ctx, p).ServeHTTP(w, r)
		})
		h.HandleFunc("/api/v1/htbt/devices", func(w http.ResponseWriter, r *http.Request) {
			var b interface{}
			// var err error // Commenting this out due to shadowing over L118
			if r.Method == "GET" {
				b = map[string]interface{}{
					"code":    0,
					"devices": v.devices,
				}
			} else {
				var o []byte
				if o, err = ioutil.ReadAll(r.Body); err == nil {
					obj := struct {
						ID   string      `json:"id"`
						Data interface{} `json:"data"`
					}{}
					if err = json.Unmarshal(o, &obj); err == nil {
						v.devices[obj.ID] = obj.Data
						b = map[string]int{
							"code": 0,
						}
					}
				}
			}

			if err != nil {
				ocore.Error(ctx, err).ServeHTTP(w, r)
			} else {
				ocore.Data(ctx, b).ServeHTTP(w, r)
			}
		})
		if err = http.Serve(l, h); err != nil {
			if !core.IsNormalQuit(err) && !isListenerClosed {
				core.Error.Println(ctx, "htbt(api) serve failed. err is", err)
			}
			return
		}
		core.Trace.Println(ctx, "htbt(api) terminated.")
	})

	// should quit?
	w.GFork("", func(wc core.WorkerContainer) {
		<-wc.QC()
		defer wc.Quit()
		isListenerClosed = true
		_ = l.Close()
	})

	return
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
				continue
			}

			if len(v.ips) <= 0 {
				interval = discoveryEmptyInterval
				continue
			}
			core.Trace.Println(ctx, "local ip is", v.ips, "exported", v.exportIP)
			interval = discoveryRefreshInterval
		}
	}

	// This is unreachable
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
				core.Warn.Println(ctx, "heartbeat to", c.URL, "every", c.Interval, "failed, err is", err)
			} else {
				core.Info.Println(ctx, "heartbeat to", c.URL, "every", c.Interval)
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

	// whether address is internet.
	isInternet := func(ipv4 net.IP) IfaceType {
		ipv4 = ipv4.To4()
		core.Info.Println(ctx, "addr is", []byte(ipv4))
		addr := (uint32(ipv4[0]) << 24) | (uint32(ipv4[1]) << 16) | (uint32(ipv4[2]) << 8) | uint32(ipv4[3])

		// lo, 127.0.0.0-127.0.0.1
		if addr >= 0x7f000000 && addr <= 0x7f000001 {
			return IfaceIntranet
		}

		// Class A 10.0.0.0-10.255.255.255
		if addr >= 0x0a000000 && addr <= 0x0affffff {
			return IfaceIntranet
		}

		// Class B 172.16.0.0-172.31.255.255
		if addr >= 0xac100000 && addr <= 0xac1fffff {
			return IfaceIntranet
		}

		// Class C 192.168.0.0-192.168.255.255
		if addr >= 0xc0a80000 && addr <= 0xc0a8ffff {
			return IfaceIntranet
		}

		return IfaceInternet
	}

	// fetch the ip from addr interface.
	ipf := func(addr net.Addr) (string, bool, IfaceType) {
		if v, ok := addr.(*net.IPNet); ok && vf(v.IP) {
			return v.IP.String(), true, isInternet(v.IP)
		} else if v, ok := addr.(*net.IPAddr); ok && vf(v.IP) {
			return v.IP.String(), true, isInternet(v.IP)
		} else {
			return "", false, IfaceUnknown
		}
	}

	var ifaces []net.Interface
	if ifaces, err = net.Interfaces(); err != nil {
		return
	}

	v.exportIP = nil
	v.ips = make([]*NetworkIface, 0)
	for _, iface := range ifaces {
		// ignore any loopback interface.
		if (iface.Flags & net.FlagLoopback) == net.FlagLoopback {
			continue
		}

		var addrs []net.Addr
		if addrs, err = iface.Addrs(); err != nil {
			return
		}
		if len(addrs) == 0 {
			continue
		}
		core.Info.Println(ctx, "scan iface", iface.Name, "flags", iface.Flags, "addrs", addrs, "hwaddr", iface.HardwareAddr)

		// dumps all network interfaces.
		for _, addr := range addrs {
			if p, ok, pub := ipf(addr); ok {
				core.Trace.Println(ctx, fmt.Sprintf("match iface=%v, ip=%v, hwaddr=%v, pub=%v", iface.Name, p, iface.HardwareAddr, pub))
				v.ips = append(v.ips, &NetworkIface{
					Ifname: iface.Name, IP: p, Mac: iface.HardwareAddr.String(), Internet: pub,
				})
			} else {
				core.Info.Println(ctx, "iface", iface.Name, addr, reflect.TypeOf(addr))
			}
		}
	}

	// find the best match public address.
	for _, ip := range v.ips {
		if ip.Internet == IfaceInternet {
			v.exportIP = ip
			return
		}
	}

	// no public address, use private address.
	if len(v.ips) > 0 {
		v.exportIP = v.ips[core.Conf.Stat.Network%len(v.ips)]
	}
	return
}

func (v *Heartbeat) beat() (err error) {
	ctx := v.ctx

	v.lock.Lock()
	defer v.lock.Unlock()

	if v.exportIP == nil {
		core.Info.Println(ctx, "heartbeat not ready.")
		return
	}

	p := struct {
		DeviceID string      `json:"device_id"`
		IP       string      `json:"ip"`
		Summary  interface{} `json:"summaries,omitempty"`
		Devices  interface{} `json:"devices,omitempty"`
	}{}

	c := &core.Conf.Heartbeat
	p.DeviceID = c.DeviceID
	p.IP = v.exportIP.IP
	if len(v.devices) > 0 {
		p.Devices = v.devices
	}

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
	if resp, err = http.Post(c.URL, ocore.HttpJson, bytes.NewReader(b)); err != nil {
		return
	}
	defer resp.Body.Close()

	core.Info.Println(ctx, "heartbeat to", c.URL, "ok")
	return
}
