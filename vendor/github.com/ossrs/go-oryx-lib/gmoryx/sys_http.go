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

// The gmoryx(gomobile oryx APIs) http package exported for mobile.
package gmoryx

import (
	"fmt"
	"net"
	"net/http"
)

type HttpResponseWriter interface {
	Write([]byte) (int, error)
}

type HttpRequest struct {
	r *http.Request
}

type HttpHandler interface {
	ServeHTTP(HttpResponseWriter, *HttpRequest)
}

func HttpHandle(pattern string, handler HttpHandler) {
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, &HttpRequest{r: r})
	})
}

var httpError error
var httpListener net.Listener

func HttpListenAndServe(addr string, handler HttpHandler) (err error) {
	var h http.Handler
	if handler != nil {
		h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, &HttpRequest{r: r})
		})
	}

	srv := &http.Server{Addr: addr, Handler: h}

	if addr = srv.Addr; addr == "" {
		addr = ":http"
	}

	httpListener, err = net.Listen("tcp", addr)
	if err != nil {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				httpError = fmt.Errorf("Recover from %v", r)
			}
		}()
		srv.Serve(httpListener.(*net.TCPListener))
	}()

	return
}

func HttpShutdown() (err error) {
	if httpListener != nil {
		err = httpListener.Close()
		httpListener = nil
	}

	if err != nil {
		return
	}

	return httpError
}
