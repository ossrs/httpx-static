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
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/ossrs/go-oryx-lib/https"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
)

const server = "Oryx/0.0.2"

func main() {
	fmt.Println(server, "HTTP/HTTPS static server with API proxy.")

	var err error
	var httpPort, httpsPort int
	var httpsDomains, html, cacheFile string
	var useLetsEncrypt bool
	var ssCert, ssKey string
	var oproxy string
	flag.IntVar(&httpPort, "http", 0, "http listen at. 0 to disable http.")
	flag.IntVar(&httpsPort, "https", 0, "https listen at. 0 to disable https. 443 to serve. ")
	flag.StringVar(&httpsDomains, "domains", "", "the allow domains, empty to allow all. for example: ossrs.net,www.ossrs.net")
	flag.StringVar(&html, "root", "./html", "the www web root. support relative dir to argv[0].")
	flag.StringVar(&cacheFile, "cache", "./letsencrypt.cache", "the cache for https. support relative dir to argv[0].")
	flag.BoolVar(&useLetsEncrypt, "lets", false, "whether use letsencrypt CA. self sign if not.")
	flag.StringVar(&ssKey, "ssk", "server.key", "https self-sign key by(before server.cert): openssl genrsa -out server.key 2048")
	flag.StringVar(&ssCert, "ssc", "server.crt", "https self-sign cert by: openssl req -new -x509 -key server.key -out server.crt -days 365")
	flag.StringVar(&oproxy, "proxy", "", "proxy the matched path to backend, for example, -proxy http://127.0.0.1:8888/api/webrtc")
	flag.Parse()

	if httpsPort != 0 && httpsPort != 443 {
		fmt.Println("https must be 0(disabled) or 443(enabled)")
		os.Exit(-1)
	}
	if httpPort == 0 && httpsPort == 0 {
		flag.PrintDefaults()
		os.Exit(-1)
	}

	var proxyUrl *url.URL
	var proxy *httputil.ReverseProxy
	if oproxy != "" {
		if proxyUrl, err = url.Parse(oproxy); err != nil {
			fmt.Println("proxy is not legal url, proxy is", oproxy)
			os.Exit(-1)
		}

		proxy = &httputil.ReverseProxy{
			Director: func(r *http.Request) {
				r.URL.Scheme = proxyUrl.Scheme
				r.URL.Host = proxyUrl.Host
				if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
					r.Header.Set("X-Real-IP", ip)
				}
				//fmt.Println(fmt.Sprintf("proxy http %v to %v", r.RemoteAddr, r.URL.String()))
			},
		}
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

		if proxyUrl != nil && strings.HasPrefix(r.URL.Path, proxyUrl.Path) {
			// For matched OPTIONS, directly return without response.
			if r.Method == "OPTIONS" {
				return
			}

			proxy.ServeHTTP(w, r)
			return
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
	fmt.Println(fmt.Sprintf("%v html root at %v", strings.Join(protos, ", "), string(html)))

	wg := sync.WaitGroup{}
	go func() {
		defer wg.Done()

		if httpPort == 0 {
			fmt.Println("http server disabled")
			return
		}

		if err := http.ListenAndServe(fmt.Sprintf(":%v", httpPort), nil); err != nil {
			panic(err)
		}
		fmt.Println("http server ok.")
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()

		if httpsPort == 0 {
			fmt.Println("https server disabled")
			return
		}

		var m https.Manager

		if useLetsEncrypt {
			var domains []string
			if httpsDomains != "" {
				domains = strings.Split(httpsDomains, ",")
			}

			if m, err = https.NewLetsencryptManager("", domains, cacheFile); err != nil {
				panic(err)
			}
		} else {
			if m, err = https.NewSelfSignManager(ssCert, ssKey); err != nil {
				panic(err)
			}
		}

		svr := &http.Server{
			Addr: fmt.Sprintf(":%v", httpsPort),
			TLSConfig: &tls.Config{
				GetCertificate: m.GetCertificate,
			},
		}

		if err := svr.ListenAndServeTLS("", ""); err != nil {
			panic(err)
		}
		fmt.Println("https server ok.")
	}()
	wg.Add(1)

	wg.Wait()
}
