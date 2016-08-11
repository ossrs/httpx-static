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
 This the main entrance of rtmplb, load-balance for rtmp streaming.
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
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

var signature = fmt.Sprintf("RTMPLB/%v", kernel.Version())

// The config object for rtmplb module.
type RtmpLbConfig struct {
	kernel.Config
	Api  string `json:"api"`
	Rtmp struct {
		Listens []string `json:"listens"`
	} `json:"rtmp"`
}

func (v *RtmpLbConfig) String() string {
	return fmt.Sprintf("%v, api=%v, rtmp(listen=%v)", &v.Config, v.Api, v.Rtmp.Listens)
}

func (v *RtmpLbConfig) Loads(c string) (err error) {
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
		return fmt.Errorf("no api")
	}

	if r := &v.Rtmp; len(r.Listens) == 0 {
		return fmt.Errorf("no rtmp listens")
	}

	return
}

func main() {
	var err error
	confFile := oo.ParseArgv("../conf/rtmplb.json", kernel.Version(), signature)
	fmt.Println("RTMPLB is the load-balance for rtmp streaming, config is", confFile)

	conf := &RtmpLbConfig{}
	if err = conf.Loads(confFile); err != nil {
		ol.E(nil, "Loads config failed, err is", err)
		return
	}
	defer conf.Close()

	ctx := &kernel.Context{}
	ol.T(ctx, fmt.Sprintf("Config ok, %v", conf))

	// rtmplb is a asprocess of shell.
	asq := make(chan bool, 1)
	oa.WatchNoExit(ctx, oa.Interval, asq)

	var listener *kernel.TcpListeners
	if listener, err = kernel.NewTcpListeners(conf.Rtmp.Listens); err != nil {
		ol.E(ctx, "create listener failed, err is", err)
		return
	}
	defer listener.Close()

	if err = listener.ListenTCP(); err != nil {
		ol.E(ctx, "listen tcp failed, err is", err)
		return
	}

	var apiListener net.Listener
	apiAddr := strings.Split(conf.Api, "://")[1]
	if apiListener, err = net.Listen("tcp", apiAddr); err != nil {
		ol.E(ctx, "http listen failed, err is", err)
		return
	}
	defer apiListener.Close()

	closing := make(chan bool, 1)
	wait := &sync.WaitGroup{}

	// rtmp connections
	go func() {
		wait.Add(1)
		defer wait.Done()

		defer func() {
			select {
			case closing <- true:
			default:
			}
		}()

		defer ol.E(ctx, "rtmp accepter ok")

		defer func() {
			listener.Close()
		}()

		for {
			var c *net.TCPConn
			if c, err = listener.AcceptTCP(); err != nil {
				if err != kernel.ListenerDisposed {
					ol.E(ctx, "accept failed, err is", err)
				}
				break
			}
			c.Close()
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

		http.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
			oh.WriteVersion(w, r, kernel.Version())
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
	listener.Close()
	apiListener.Close()
	wait.Wait()

	ol.T(ctx, "serve ok")
	return
}
