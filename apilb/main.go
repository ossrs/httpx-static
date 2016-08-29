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
 This the main entrance of apilb, load-balance for backend api.
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
	"strconv"
	"strings"
	"syscall"
)

var signature = fmt.Sprintf("APILB/%v", kernel.Version())

// The config object for apilb module.
type ApiLbConfig struct {
	kernel.Config
	Api     string `json:"api"`
	Backend struct {
		Enabled bool   `json:"enabled"`
		Api     string `json:"api"`
	} `json:"backend"`
}

func (v *ApiLbConfig) String() string {
	return fmt.Sprintf("%v, api=%v, backend(%v,api=%v)",
		&v.Config, v.Api, v.Backend.Enabled, v.Backend.Api)
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

	if v.Backend.Enabled && len(v.Backend.Api) == 0 {
		return fmt.Errorf("Empty backend api")
	}

	return
}

const (
	Success     oh.SystemError = 0
	ApiBackendError oh.SystemError = 100 + iota
)

// The http proxy for backend api.
type proxy struct {
	conf         *ApiLbConfig
	backendPorts []int
	backendPort  int
	rp           *httputil.ReverseProxy
}

func NewProxy(conf *ApiLbConfig) *proxy {
	v := &proxy{conf: conf}
	v.rp = &httputil.ReverseProxy{Director: nil}
	return v
}

func (v *proxy) serveControl(ctx ol.Context, r *http.Request) (string, oh.SystemError) {
	var err error
	var port int

	q := r.URL.Query()
	if value := q.Get("port"); len(value) == 0 {
		return "miss port", ApiBackendError
	} else if port, err = strconv.Atoi(value); err != nil {
		return fmt.Sprintf("port is not int, err is %v", err), ApiBackendError
	}

	hasProxyed := func(port int, ports []int) bool {
		for _, p := range ports {
			if p == port {
				return true
			}
		}
		return false
	}

	ol.T(ctx, fmt.Sprintf("proxy backend to %v, previous=%v, ports=%v", port, v.backendPort, v.backendPorts))
	if !hasProxyed(port, v.backendPorts) {
		v.backendPorts = append(v.backendPorts, port)
	}
	v.backendPort = port

	return "", Success
}

func (v *proxy) serveBackendApi(w http.ResponseWriter, r *http.Request) {
	ctx := &kernel.Context{}

	v.rp.Director = func(r *http.Request) {
		r.URL.Scheme = "http"

		r.URL.Host = fmt.Sprintf("127.0.0.1:%v", v.backendPort)
		if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			r.Header.Set("X-Real-IP", ip)
		}
		ol.W(ctx, fmt.Sprintf("proxy backend %v to %v", r.RemoteAddr, r.URL.String()))
	}

	v.rp.ServeHTTP(w, r)
}

func main() {
	var err error

	// for shell.
	var api, backend string
	flag.StringVar(&api, "api", "", "The api tcp://host:port, optional.")
	flag.StringVar(&backend, "backend", "", "The backend api tcp://host:port, optional.")

	confFile := oo.ParseArgv("../conf/apilb.json", kernel.Version(), signature)
	fmt.Println("APILB is the load-balance for http api, config is", confFile)

	conf := &ApiLbConfig{}
	if err = conf.Loads(confFile); err != nil {
		ol.E(nil, "Loads config failed, err is", err)
		return
	}
	defer conf.Close()

	// override by shell.
	if len(backend) > 0 {
		conf.Backend.Enabled, conf.Backend.Api = true, backend
	}
	if len(api) > 0 {
		conf.Api = api
	}

	ctx := &kernel.Context{}
	ol.T(ctx, fmt.Sprintf("Config ok, %v", conf))

	// httplb is a asprocess of shell.
	asq := make(chan bool, 1)
	oa.WatchNoExit(ctx, oa.Interval, asq)

	var backendListener net.Listener
	var backendNetwork, backendAddr string
	if conf.Backend.Enabled {
		addrs := strings.Split(conf.Backend.Api, "://")
		backendNetwork, backendAddr = addrs[0], addrs[1]
		if backendListener, err = net.Listen(backendNetwork, backendAddr); err != nil {
			ol.E(ctx, "backend api listen failed, err is", err)
			return
		}
		defer backendListener.Close()
	}

	var apiListener net.Listener
	addrs := strings.Split(conf.Api, "://")
	apiNetwork, apiAddr := addrs[0], addrs[1]
	if apiListener, err = net.Listen(apiNetwork, apiAddr); err != nil {
		ol.E(ctx, "api listen failed, err is", err)
		return
	}
	defer apiListener.Close()

	proxy := NewProxy(conf)
	oh.Server = signature

	wg := kernel.NewWorkerGroup()
	defer ol.T(ctx, "serve ok")
	defer wg.Close()

	wg.QuitForChan(asq)
	wg.QuitForSignals(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	// backend api proxy.
	if conf.Backend.Enabled {
		wg.ForkGoroutine(func() {
			ol.E(ctx, "backend api porxy ready")
			defer ol.E(ctx, "backend api proxy ok")

			handler := http.NewServeMux()

			ol.T(ctx, fmt.Sprintf("handle http://%v/", backendAddr))
			handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				proxy.serveBackendApi(w, r)
			})

			server := &http.Server{Addr: backendNetwork, Handler: handler}
			if err = server.Serve(backendListener); err != nil {
				if !wg.Closed() {
					ol.E(ctx, "backend api serve failed, err is", err)
				}
				return
			}
		}, func() {
			backendListener.Close()
		})
	}

	// control messages
	wg.ForkGoroutine(func() {
		ol.E(ctx, "api handler ready")
		defer ol.E(ctx, "api handler ok")

		handler := http.NewServeMux()

		ol.T(ctx, fmt.Sprintf("handle http://%v/api/v1/version", apiAddr))
		handler.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
			oh.WriteVersion(w, r, kernel.Version())
		})

		ol.T(ctx, fmt.Sprintf("handle http://%v/api/v1/proxy?port=19850", apiAddr))
		handler.HandleFunc("/api/v1/proxy", func(w http.ResponseWriter, r *http.Request) {
			ctx := &kernel.Context{}
			if msg, err := proxy.serveControl(ctx, r); err != Success {
				oh.WriteCplxError(ctx, w, r, err, msg)
				return
			}
			oh.WriteData(ctx, w, r, nil)
		})

		server := &http.Server{Addr: apiAddr, Handler: handler}
		if err = server.Serve(apiListener); err != nil {
			if !wg.Closed() {
				ol.E(ctx, "api serve failed, err is", err)
			}
			return
		}
	}, func() {
		apiListener.Close()
	})

	wg.Wait()
	return
}
