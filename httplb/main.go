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
 This the main entrance of httplb, load-balance for flv/hls+ streaming.
*/
package main

import (
	"encoding/json"
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

var signature = fmt.Sprintf("HTTPLB/%v", kernel.Version())

// The config object for httplb module.
type HttpLbConfig struct {
	kernel.Config
	Api  string `json:"api"`
	Http struct {
		Listen string `json:"listen"`
	} `json:"http"`
}

func (v *HttpLbConfig) String() string {
	return fmt.Sprintf("%v", &v.Config)
}

func (v *HttpLbConfig) Loads(c string) (err error) {
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
	} else if nn := strings.Count(v.Api, "://"); nn != 1 {
		return fmt.Errorf("Api contains %v network", nn)
	}

	if len(v.Http.Listen) == 0 {
		return fmt.Errorf("Empty http listens")
	}
	if nn := strings.Count(v.Http.Listen, "://"); nn != 1 {
		return fmt.Errorf("Listen %v contains %v network", v.Http.Listen, nn)
	}

	return
}

type proxy struct {
	conf       *HttpLbConfig
	ports      []int
	activePort int
	rp         *httputil.ReverseProxy
}

func NewProxy(conf *HttpLbConfig) *proxy {
	v := &proxy{conf: conf}

	director := func(r *http.Request) {
		r.URL.Scheme = "http"
		from := r.URL.String()

		h, _, _ := net.SplitHostPort(r.URL.Host)
		r.URL.Host = fmt.Sprintf("%v:%v", h, v.activePort)
		to := r.URL.String()

		ol.W(nil, "proxy", from, "to", to)
	}
	v.rp = &httputil.ReverseProxy{Director: director}

	return v
}

func hasAnySuffixes(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

func (v *proxy) serveHttp(w http.ResponseWriter, r *http.Request) {
	ctx := &kernel.Context{}

	if v.activePort <= 0 {
		oh.Error(ctx, fmt.Errorf("Backend not ready")).ServeHTTP(w, r)
		return
	}

	if p := r.URL.Path; !hasAnySuffixes(p, ".flv", ".m3u8", ".ts", ".xml") {
		http.NotFound(w, r)
		return
	}

	v.rp.ServeHTTP(w, r)
}

const (
	Success       oh.SystemError = 0
	ApiProxyQuery oh.SystemError = 100 + iota
)

func (v *proxy) serveChangeBackendApi(r *http.Request) (string, oh.SystemError) {
	var err error
	q := r.URL.Query()
	ctx := &kernel.Context{}

	var httpPort string
	if httpPort = q.Get("http"); len(httpPort) == 0 {
		return fmt.Sprintf("require query http port"), ApiProxyQuery
	}

	var port int
	if port, err = strconv.Atoi(httpPort); err != nil {
		return fmt.Sprintf("http port is not int, err is %v", err), ApiProxyQuery
	}

	ol.T(ctx, fmt.Sprintf("proxy http to %v, previous=%v, ports=%v", port, v.activePort, v.ports))
	if !v.hasProxyed(port) {
		v.ports = append(v.ports, port)
	}
	v.activePort = port

	return "", Success
}

func (v *proxy) hasProxyed(port int) bool {
	for _, p := range v.ports {
		if p == port {
			return true
		}
	}
	return false
}

func main() {
	var err error
	confFile := oo.ParseArgv("../conf/httplb.json", kernel.Version(), signature)
	fmt.Println("HTTPLB is the load-balance for http flv/hls+ streaming, config is", confFile)

	conf := &HttpLbConfig{}
	if err = conf.Loads(confFile); err != nil {
		ol.E(nil, "Loads config failed, err is", err)
		return
	}
	defer conf.Close()

	ctx := &kernel.Context{}
	ol.T(ctx, fmt.Sprintf("Config ok, %v", conf))

	// httplb is a asprocess of shell.
	asq := make(chan bool, 1)
	oa.WatchNoExit(ctx, oa.Interval, asq)

	var httpListener net.Listener
	addrs := strings.Split(conf.Http.Listen, "://")
	httpNetwork, httpAddr := addrs[0], addrs[1]
	if httpListener, err = net.Listen(httpNetwork, httpAddr); err != nil {
		ol.E(ctx, "http listen failed, err is", err)
		return
	}
	defer httpListener.Close()

	var apiListener net.Listener
	addrs = strings.Split(conf.Api, "://")
	apiNetwork, apiAddr := addrs[0], addrs[1]
	if apiListener, err = net.Listen(apiNetwork, apiAddr); err != nil {
		ol.E(ctx, "http listen failed, err is", err)
		return
	}
	defer apiListener.Close()

	closing := make(chan bool, 1)
	wait := &sync.WaitGroup{}
	proxy := NewProxy(conf)

	oh.Server = signature

	// http proxy.
	go func() {
		wait.Add(1)
		defer wait.Done()

		defer func() {
			select {
			case closing <- true:
			default:
			}
		}()

		defer ol.E(ctx, "http proxy ok")

		ol.T(ctx, fmt.Sprintf("handle http://%v/", httpAddr))
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			proxy.serveHttp(w, r)
		})

		server := &http.Server{Addr: httpNetwork, Handler: nil}
		if err = server.Serve(httpListener); err != nil {
			ol.E(ctx, "http serve failed, err is", err)
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

		defer ol.E(ctx, "http handler ok")

		oh.Server = signature

		ol.T(ctx, fmt.Sprintf("handle http://%v/api/v1/version", apiAddr))
		http.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
			oh.WriteVersion(w, r, kernel.Version())
		})

		ol.T(ctx, fmt.Sprintf("handle http://%v/api/v1/proxy?http=8081", apiAddr))
		http.HandleFunc("/api/v1/proxy", func(w http.ResponseWriter, r *http.Request) {
			if msg, err := proxy.serveChangeBackendApi(r); err != Success {
				oh.CplxError(ctx, err, msg).ServeHTTP(w, r)
				return
			}
			oh.Data(ctx, nil).ServeHTTP(w, r)
		})

		server := &http.Server{Addr: apiAddr, Handler: nil}
		if err = server.Serve(apiListener); err != nil {
			ol.E(ctx, "http serve failed, err is", err)
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
	httpListener.Close()
	apiListener.Close()
	wait.Wait()

	ol.T(ctx, "serve ok")
	return
}
