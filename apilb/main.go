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
 This the main entrance of apilb, load-balance for srs/big api.
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	oa "github.com/ossrs/go-oryx-lib/asprocess"
	oh "github.com/ossrs/go-oryx-lib/http"
	oj "github.com/ossrs/go-oryx-lib/json"
	ol "github.com/ossrs/go-oryx-lib/logger"
	oo "github.com/ossrs/go-oryx-lib/options"
	"github.com/ossrs/go-oryx/kernel"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

var signature = fmt.Sprintf("APILB/%v", kernel.Version())

// The config object for apilb module.
type ApiLbConfig struct {
	kernel.Config
	Api string `json:"api"`
	Srs struct {
		Enabled bool   `json:"enabled"`
		Api     string `json:"api"`
	} `json:"srs"`
	Big struct {
		Enabled bool   `json:"enabled"`
		Api     string `json:"api"`
	} `json:"big"`
}

func (v *ApiLbConfig) String() string {
	return fmt.Sprintf("%v, api=%v, srs(%v,api=%v), big(%v,api=%v)",
		&v.Config, v.Api, v.Srs.Enabled, v.Srs.Api, v.Big.Enabled, v.Big.Api)
}

func (v *ApiLbConfig) Loads(c string) (err error) {
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

	if err = v.Config.OpenLogger(); err != nil {
		ol.E(nil, "Open logger failed, err is", err)
		return
	}

	if len(v.Api) == 0 {
		return fmt.Errorf("Empty api")
	}

	if v.Srs.Enabled && len(v.Srs.Api) == 0 {
		return fmt.Errorf("Empty srs api")
	}

	if v.Big.Enabled && len(v.Big.Api) == 0 {
		return fmt.Errorf("Empty big api")
	}

	return
}

const (
	Success     oh.SystemError = 0
	ApiSrsError oh.SystemError = 100 + iota
	ApiBigError
)

// The http proxy for backend api.
type proxy struct {
	conf     *ApiLbConfig
	srsPorts []int
	srsPort  int
	bigPorts []int
	bigPort  int
	rp       *httputil.ReverseProxy
}

func NewProxy(conf *ApiLbConfig) *proxy {
	v := &proxy{conf: conf}
	v.rp = &httputil.ReverseProxy{Director: nil}
	return v
}

func (v *proxy) serveControlSrs(ctx ol.Context, r *http.Request) (string, oh.SystemError) {
	var err error
	var port int

	q := r.URL.Query()
	if value := q.Get("port"); len(value) == 0 {
		return "miss port", ApiSrsError
	} else if port, err = strconv.Atoi(value); err != nil {
		return fmt.Sprintf("port is not int, err is %v", err), ApiSrsError
	}

	hasProxyed := func(port int, ports []int) bool {
		for _, p := range ports {
			if p == port {
				return true
			}
		}
		return false
	}

	ol.T(ctx, fmt.Sprintf("proxy srs to %v, previous=%v, ports=%v", port, v.srsPort, v.srsPorts))
	if !hasProxyed(port, v.srsPorts) {
		v.srsPorts = append(v.srsPorts, port)
	}
	v.srsPort = port

	return "", Success
}

func (v *proxy) serveControlBig(ctx ol.Context, r *http.Request) (string, oh.SystemError) {
	var err error
	var port int

	q := r.URL.Query()
	if value := q.Get("port"); len(value) == 0 {
		return "miss port", ApiSrsError
	} else if port, err = strconv.Atoi(value); err != nil {
		return fmt.Sprintf("port is not int, err is %v", err), ApiSrsError
	}

	hasProxyed := func(port int, ports []int) bool {
		for _, p := range ports {
			if p == port {
				return true
			}
		}
		return false
	}

	ol.T(ctx, fmt.Sprintf("proxy big to %v, previous=%v, ports=%v", port, v.bigPort, v.bigPorts))
	if !hasProxyed(port, v.bigPorts) {
		v.bigPorts = append(v.bigPorts, port)
	}
	v.bigPort = port

	return "", Success
}

func (v *proxy) serveSrsApi(w http.ResponseWriter, r *http.Request) {
	ctx := &kernel.Context{}

	v.rp.Director = func(r *http.Request) {
		r.URL.Scheme = "http"

		r.URL.Host = fmt.Sprintf("127.0.0.1:%v", v.srsPort)
		if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			r.Header.Set("X-Real-IP", ip)
		}
		ol.W(ctx, fmt.Sprintf("proxy srs %v to %v", r.RemoteAddr, r.URL.String()))
	}

	v.rp.ServeHTTP(w, r)
}

func (v *proxy) serveBigApi(w http.ResponseWriter, r *http.Request) {
	ctx := &kernel.Context{}

	v.rp.Director = func(r *http.Request) {
		r.URL.Scheme = "http"

		r.URL.Host = fmt.Sprintf("127.0.0.1:%v", v.bigPort)
		if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			r.Header.Set("X-Real-IP", ip)
		}
		ol.W(ctx, fmt.Sprintf("proxy big %v to %v", r.RemoteAddr, r.URL.String()))
	}

	v.rp.ServeHTTP(w, r)
}

func main() {
	var err error

	// for shell.
	var api, srs, big string
	flag.StringVar(&api, "api", "", "The api tcp://host:port, optional.")
	flag.StringVar(&srs, "srs", "", "The srs api tcp://host:port, optional.")
	flag.StringVar(&big, "big", "", "The big api tcp://host:port, optional.")

	confFile := oo.ParseArgv("../conf/apilb.json", kernel.Version(), signature)
	fmt.Println("APILB is the load-balance for http api, config is", confFile)

	conf := &ApiLbConfig{}
	if err = conf.Loads(confFile); err != nil {
		ol.E(nil, "Loads config failed, err is", err)
		return
	}
	defer conf.Close()

	// override by shell.
	if len(srs) > 0 {
		conf.Srs.Enabled, conf.Srs.Api = true, srs
	}
	if len(big) > 0 {
		conf.Big.Enabled, conf.Big.Api = true, big
	}
	if len(api) > 0 {
		conf.Api = api
	}

	ctx := &kernel.Context{}
	ol.T(ctx, fmt.Sprintf("Config ok, %v", conf))

	// httplb is a asprocess of shell.
	asq := make(chan bool, 1)
	oa.WatchNoExit(ctx, oa.Interval, asq)

	var srsListener net.Listener
	var srsNetwork, srsAddr string
	if conf.Srs.Enabled {
		addrs := strings.Split(conf.Srs.Api, "://")
		srsNetwork, srsAddr = addrs[0], addrs[1]
		if srsListener, err = net.Listen(srsNetwork, srsAddr); err != nil {
			ol.E(ctx, "srs api listen failed, err is", err)
			return
		}
		defer srsListener.Close()
	}

	var bigListener net.Listener
	var bigNetwork, bigAddr string
	if conf.Big.Enabled {
		addrs := strings.Split(conf.Big.Api, "://")
		bigNetwork, bigAddr = addrs[0], addrs[1]
		if bigListener, err = net.Listen(bigNetwork, bigAddr); err != nil {
			ol.E(ctx, "big api listen failed, err is", err)
			return
		}
		defer bigListener.Close()
	}

	var apiListener net.Listener
	addrs := strings.Split(conf.Api, "://")
	apiNetwork, apiAddr := addrs[0], addrs[1]
	if apiListener, err = net.Listen(apiNetwork, apiAddr); err != nil {
		ol.E(ctx, "api listen failed, err is", err)
		return
	}
	defer apiListener.Close()

	closing := make(chan bool, 1)
	wait := &sync.WaitGroup{}
	proxy := NewProxy(conf)

	oh.Server = signature

	// srs api proxy.
	go func() {
		if !conf.Srs.Enabled {
			return
		}

		wait.Add(1)
		defer wait.Done()

		defer func() {
			select {
			case closing <- true:
			default:
			}
		}()

		defer ol.E(ctx, "srs api proxy ok")

		handler := http.NewServeMux()

		ol.T(ctx, fmt.Sprintf("handle http://%v/", srsAddr))
		handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			proxy.serveSrsApi(w, r)
		})

		server := &http.Server{Addr: srsNetwork, Handler: handler}
		if err = server.Serve(srsListener); err != nil {
			ol.E(ctx, "srs api serve failed, err is", err)
			return
		}
	}()

	// big api proxy
	go func() {
		if !conf.Big.Enabled {
			return
		}

		wait.Add(1)
		defer wait.Done()

		defer func() {
			select {
			case closing <- true:
			default:
			}
		}()

		defer ol.E(ctx, "big api handler ok")

		handler := http.NewServeMux()

		ol.T(ctx, fmt.Sprintf("handle http://%v/", bigAddr))
		handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			proxy.serveBigApi(w, r)
		})

		server := &http.Server{Addr: bigAddr, Handler: handler}
		if err = server.Serve(bigListener); err != nil {
			ol.E(ctx, "big api serve failed, err is", err)
			return
		}
	}()

	// control messages
	go func() {
		wait.Add(1)
		defer wait.Done()

		defer func() {
			select {
			case closing <- true:
			default:
			}
		}()

		defer ol.E(ctx, "api handler ok")

		handler := http.NewServeMux()

		ol.T(ctx, fmt.Sprintf("handle http://%v/api/v1/version", apiAddr))
		handler.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
			oh.WriteVersion(w, r, kernel.Version())
		})

		ol.T(ctx, fmt.Sprintf("handle http://%v/api/v1/proxy/srs?port=19850", apiAddr))
		handler.HandleFunc("/api/v1/proxy/srs", func(w http.ResponseWriter, r *http.Request) {
			ctx := &kernel.Context{}
			if msg, err := proxy.serveControlSrs(ctx, r); err != Success {
				oh.WriteCplxError(ctx, w, r, err, msg)
				return
			}
			oh.WriteData(ctx, w, r, nil)
		})

		ol.T(ctx, fmt.Sprintf("handle http://%v/api/v1/proxy/big?port=19900", apiAddr))
		handler.HandleFunc("/api/v1/proxy/big", func(w http.ResponseWriter, r *http.Request) {
			ctx := &kernel.Context{}
			if msg, err := proxy.serveControlBig(ctx, r); err != Success {
				oh.WriteCplxError(ctx, w, r, err, msg)
				return
			}
			oh.WriteData(ctx, w, r, nil)
		})

		server := &http.Server{Addr: apiAddr, Handler: handler}
		if err = server.Serve(apiListener); err != nil {
			ol.E(ctx, "api serve failed, err is", err)
			return
		}
	}()

	// listen singal.
	go func() {
		ss := make(chan os.Signal)
		signal.Notify(ss, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		for s := range ss {
			ol.E(ctx, "quit for signal", s)
			closing <- true
		}
	}()

	// cleanup when got closing event.
	select {
	case <-closing:
		closing <- true
	case <-asq:
	}
	srsListener.Close()
	bigListener.Close()
	apiListener.Close()
	wait.Wait()

	ol.T(ctx, "serve ok")
	return
}
