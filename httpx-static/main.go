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
 This the main entrance of https-proxy, proxy to api or other http server.
*/
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	oe "github.com/ossrs/go-oryx-lib/errors"
	"github.com/ossrs/go-oryx-lib/https"
	ol "github.com/ossrs/go-oryx-lib/logger"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
)

const server = "Oryx/0.0.3"

type Proxies []string

func (v *Proxies) String() string {
	return "proxy to backend services"
}

func (v *Proxies) Set(value string) error {
	*v = append(*v, value)
	return nil
}

func run(ctx context.Context) error {
	fmt.Println(server, "HTTP/HTTPS static server with API proxy.")

	var httpPort, httpsPort int
	var httpsDomains, html, cacheFile string
	var useLetsEncrypt bool
	var ssCert, ssKey string
	var oproxies Proxies
	flag.IntVar(&httpPort, "http", 0, "http listen at. 0 to disable http.")
	flag.IntVar(&httpsPort, "https", 0, "https listen at. 0 to disable https. 443 to serve. ")
	flag.StringVar(&httpsDomains, "domains", "", "the allow domains, empty to allow all. for example: ossrs.net,www.ossrs.net")
	flag.StringVar(&html, "root", "./html", "the www web root. support relative dir to argv[0].")
	flag.StringVar(&cacheFile, "cache", "./letsencrypt.cache", "the cache for https. support relative dir to argv[0].")
	flag.BoolVar(&useLetsEncrypt, "lets", false, "whether use letsencrypt CA. self sign if not.")
	flag.StringVar(&ssKey, "ssk", "server.key", "https self-sign key by(before server.cert): openssl genrsa -out server.key 2048")
	flag.StringVar(&ssCert, "ssc", "server.crt", `https self-sign cert by: openssl req -new -x509 -key server.key -out server.crt -days 365 -subj "/C=CN/ST=Beijing/L=Beijing/O=Me/OU=Me/CN=me.org"`)
	flag.Var(&oproxies, "proxy", "one or more proxy the matched path to backend, for example, -proxy http://127.0.0.1:8888/api/webrtc")
	flag.Parse()

	if useLetsEncrypt && (httpsPort != 0 && httpsPort != 443) {
		return oe.Errorf("for letsencrypt, https=%v must be 0(disabled) or 443(enabled)", httpsPort)
	}
	if httpPort == 0 && httpsPort == 0 {
		flag.PrintDefaults()
		os.Exit(-1)
	}

	var proxyUrls []*url.URL
	proxies := map[string]*httputil.ReverseProxy{}
	for _, oproxy := range []string(oproxies) {
		if oproxy == "" {
			return oe.Errorf("empty proxy in %v", oproxies)
		}

		proxyUrl, err := url.Parse(oproxy)
		if err != nil {
			return oe.Wrapf(err, "parse proxy %v", oproxy)
		}

		proxy := &httputil.ReverseProxy{
			Director: func(r *http.Request) {
				// about the x-real-schema, we proxy to backend to identify the client schema.
				if rschema := r.Header.Get("X-Real-Schema"); rschema == "" {
					if r.TLS == nil {
						r.Header.Set("X-Real-Schema", "http")
					} else {
						r.Header.Set("X-Real-Schema", "https")
					}
				}

				// about x-real-ip and x-forwarded-for or
				// about X-Real-IP and X-Forwarded-For or
				// https://segmentfault.com/q/1010000002409659
				// https://distinctplace.com/2014/04/23/story-behind-x-forwarded-for-and-x-real-ip-headers/
				// @remark http proxy will set the X-Forwarded-For.
				if rip := r.Header.Get("X-Real-IP"); rip == "" {
					if rip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
						r.Header.Set("X-Real-IP", rip)
					}
				}

				r.URL.Scheme = proxyUrl.Scheme
				r.URL.Host = proxyUrl.Host

				//ol.Tf(ctx, "proxy http %v to %v", r.RemoteAddr, r.URL.String())
			},
		}

		if _, ok := proxies[proxyUrl.Path]; ok {
			return oe.Errorf("proxy %v duplicated", proxyUrl.Path)
		}

		proxyUrls = append(proxyUrls, proxyUrl)
		proxies[proxyUrl.Path] = proxy
		ol.Tf(ctx, "Proxy %v to %v", proxyUrl.Path, oproxy)
	}

	if !path.IsAbs(cacheFile) && path.IsAbs(os.Args[0]) {
		cacheFile = path.Join(path.Dir(os.Args[0]), cacheFile)
	}
	if !path.IsAbs(html) && path.IsAbs(os.Args[0]) {
		html = path.Join(path.Dir(os.Args[0]), html)
	}

	fs := http.FileServer(http.Dir(html))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", server)

		if o := r.Header.Get("Origin"); len(o) > 0 {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, HEAD, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "Server,range,Content-Length,Content-Range")
			w.Header().Set("Access-Control-Allow-Headers", "origin,range,accept-encoding,referer,Cache-Control,X-Proxy-Authorization,X-Requested-With,Content-Type")
		}

		if proxyUrls == nil {
			fs.ServeHTTP(w, r)
			return
		}

		for _, proxyUrl := range proxyUrls {
			srcPath, proxyPath := r.URL.Path, proxyUrl.Path
			if !strings.HasSuffix(srcPath, "/") {
				// /api to /api/
				// /api.js to /api.js/
				// /api/100 to /api/100/
				srcPath += "/"
			}
			if !strings.HasSuffix(proxyPath, "/") {
				// /api/ to /api/
				// to match /api/ or /api/100
				// and not match /api.js/
				proxyPath += "/"
			}
			if !strings.HasPrefix(srcPath, proxyPath) {
				continue
			}

			// For matched OPTIONS, directly return without response.
			if r.Method == "OPTIONS" {
				return
			}

			if proxy, ok := proxies[proxyUrl.Path]; ok {
				proxy.ServeHTTP(w, r)
				return
			}
		}

		fs.ServeHTTP(w, r)
	})

	var protos []string
	if httpPort != 0 {
		protos = append(protos, fmt.Sprintf("http(:%v)", httpPort))
	}
	if httpsPort != 0 {
		s := httpsDomains
		if httpsDomains == "" {
			s = "all domains"
		}

		if useLetsEncrypt {
			protos = append(protos, fmt.Sprintf("https(:%v, %v, %v)", httpsPort, s, cacheFile))
		} else {
			protos = append(protos, fmt.Sprintf("https(:%v)", httpsPort))
		}

		if useLetsEncrypt {
			protos = append(protos, "letsencrypt")
		} else {
			protos = append(protos, fmt.Sprintf("self-sign(%v, %v)", ssKey, ssCert))
		}
	}
	ol.Tf(ctx, "%v html root at %v", strings.Join(protos, ", "), string(html))

	var hs, hss *http.Server

	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var err error
	go func() {
		defer wg.Done()
		if httpPort == 0 {
			ol.W(ctx, "http server disabled")
			return
		}

		defer cancel()
		hs = &http.Server{Addr: fmt.Sprintf(":%v", httpPort), Handler: nil}
		ol.Tf(ctx, "http serve at %v", httpPort)

		if err = hs.ListenAndServe(); err != nil {
			err = oe.Wrapf(err, "serve http")
			return
		}
		ol.T("http server ok")
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()
		if httpsPort == 0 {
			ol.W(ctx, "https server disabled")
			return
		}

		defer cancel()

		var m https.Manager
		if useLetsEncrypt {
			var domains []string
			if httpsDomains != "" {
				domains = strings.Split(httpsDomains, ",")
			}

			if m, err = https.NewLetsencryptManager("", domains, cacheFile); err != nil {
				err = oe.Wrapf(err, "create letsencrypt manager")
				return
			}
		} else {
			if m, err = https.NewSelfSignManager(ssCert, ssKey); err != nil {
				err = oe.Wrapf(err, "create self-sign manager")
				return
			}
		}

		hss = &http.Server{
			Addr: fmt.Sprintf(":%v", httpsPort),
			TLSConfig: &tls.Config{
				GetCertificate: m.GetCertificate,
			},
		}
		ol.Tf(ctx, "http serve at %v", httpsPort)

		if err = hss.ListenAndServeTLS("", ""); err != nil {
			err = oe.Wrapf(err, "listen and serve https")
			return
		}
		ol.T("https serve ok")
	}()
	wg.Add(1)

	select {
	case <-ctx.Done():
		if hs != nil {
			hs.Close()
		}
		if hss != nil {
			hss.Close()
		}
	}
	wg.Wait()

	return err
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		ol.Ef(ctx, "run err %+v", err)
		os.Exit(-1)
	}
}
