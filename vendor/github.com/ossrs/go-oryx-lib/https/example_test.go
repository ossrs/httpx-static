// The MIT License (MIT)
//
// Copyright (c) 2013-2017 Oryx(ossrs)
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

package https_test

import (
	"crypto/tls"
	"fmt"
	"github.com/ossrs/go-oryx-lib/https"
	"net/http"
)

// Requires golang 1.6+, because there's bug in http.Server
// to set the GetCertificate of TLSConfig.
func ExampleLetsencryptManagerHttpAndHttps() {
	http.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, HTTPS over letsencrypt~"))
	})

	go func() {
		if err := http.ListenAndServe(":http", nil); err != nil {
			fmt.Println("http serve failed, err is", err)
		}
	}()

	var err error
	var m https.Manager
	if m, err = https.NewLetsencryptManager("", []string{"winlin.cn"}, "letsencrypt.cache"); err != nil {
		fmt.Println("https failed, err is", err)
		return
	}

	// @remark only support listen at :https, for the validation use tls,
	//		https://github.com/ietf-wg-acme/acme/blob/master/draft-ietf-acme-acme.md#tls-with-server-name-indication-tls-sni
	svr := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: m.GetCertificate,
		},
	}

	if err := svr.ListenAndServeTLS("", ""); err != nil {
		fmt.Println("https serve failed, err is", err)
	}
}

// Requires golang 1.6+, because there's bug in http.Server
// to set the GetCertificate of TLSConfig.
func ExampleLetsencryptManagerHttps() {
	http.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, HTTPS over letsencrypt~"))
	})

	var err error
	var m https.Manager
	if m, err = https.NewLetsencryptManager("", []string{"winlin.cn"}, "letsencrypt.cache"); err != nil {
		fmt.Println("https failed, err is", err)
		return
	}

	// @remark only support listen at :https, for the validation use tls,
	//		https://github.com/ietf-wg-acme/acme/blob/master/draft-ietf-acme-acme.md#tls-with-server-name-indication-tls-sni
	svr := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: m.GetCertificate,
		},
	}

	if err := svr.ListenAndServeTLS("", ""); err != nil {
		fmt.Println("https serve failed, err is", err)
	}
}

// Requires golang 1.6+, because there's bug in http.Server
// to set the GetCertificate of TLSConfig.
func ExampleSelfSignHttps() {
	http.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, HTTPS~"))
	})

	// http://studygolang.com/articles/3175
	// openssl genrsa -out server.key 2048
	// openssl req -new -x509 -key server.key -out server.crt -days 365
	var err error
	var m https.Manager
	if m, err = https.NewSelfSignManager("server.crt", "server.key"); err != nil {
		fmt.Println("https failed, err is", err)
		return
	}

	svr := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: m.GetCertificate,
		},
	}

	if err := svr.ListenAndServeTLS("", ""); err != nil {
		fmt.Println("serve failed, err is", err)
	}
}

// Requires golang 1.6+, because there's bug in http.Server
// to set the GetCertificate of TLSConfig.
func ExampleSelfSignHttpAndHttps() {
	http.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, HTTP and HTTPS~"))
	})

	go func() {
		if err := http.ListenAndServe(":http", nil); err != nil {
			fmt.Println("http serve failed, err is", err)
		}
	}()

	// http://studygolang.com/articles/3175
	// openssl genrsa -out server.key 2048
	// openssl req -new -x509 -key server.key -out server.crt -days 365
	var err error
	var m https.Manager
	if m, err = https.NewSelfSignManager("server.crt", "server.key"); err != nil {
		fmt.Println("https failed, err is", err)
		return
	}

	svr := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: m.GetCertificate,
		},
	}

	if err := svr.ListenAndServeTLS("", ""); err != nil {
		fmt.Println("https serve failed, err is", err)
	}
}
