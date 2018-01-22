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

package http_test

import (
	"fmt"
	oh "github.com/ossrs/go-oryx-lib/http"
	"net/http"
)

func ExampleHttpTest_Global() {
	oh.Server = "Test"
	fmt.Println("Server:", oh.Server)

	// Output:
	// Server: Test
}

func ExampleHttpTest_RawResponse() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Set the common response header when need to write RAW message.
		oh.SetHeader(w)

		// Write RAW message only, or should use the Error() or Data() functions.
		w.Write([]byte("Hello, World!"))
	})
}

func ExampleHttpTest_JsonData() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Response data which can be marshal to json.
		oh.Data(nil, map[string]interface{}{
			"version": "1.0",
			"count":   100,
		}).ServeHTTP(w, r)
	})
}

func ExampleHttpTest_Error() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Response unknown error with HTTP/500
		oh.Error(nil, fmt.Errorf("System error")).ServeHTTP(w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Response known error {code:xx}
		oh.Error(nil, oh.SystemError(100)).ServeHTTP(w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Response known complex error {code:xx,data:"xxx"}
		oh.Error(nil, oh.SystemComplexError{oh.SystemError(100), "Error description"}).ServeHTTP(w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Response known complex error {code:xx,data:"xxx"}
		oh.CplxError(nil, oh.SystemError(100), "Error description").ServeHTTP(w, r)
	})
}

func ExampleWrite() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only response a success json.
		oh.Success(nil, w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Response data which can be marshal to json.
		oh.WriteData(nil, w, r, map[string]interface{}{
			"version": "1.0",
			"count":   100,
		})
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Response unknown error with HTTP/500
		oh.WriteError(nil, w, r, fmt.Errorf("System error"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Response known error {code:xx}
		oh.WriteError(nil, w, r, oh.SystemError(100))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Response known complex error {code:xx,data:"xxx"}
		oh.WriteError(nil, w, r, oh.SystemComplexError{oh.SystemError(100), "Error description"})
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Response known complex error {code:xx,data:"xxx"}
		oh.WriteCplxError(nil, w, r, oh.SystemError(100), "Error description")
	})
}

func ExampleWriteVersion() {
	http.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		// version is major.minor.revisoin-extra
		oh.WriteVersion(w, r, "1.2.3-4")
	})
}

func ExampleApiRequest() {
	var err error
	var body []byte
	if _, body, err = oh.ApiRequest("http://127.0.0.1985/api/v1/versions"); err != nil {
		return
	}

	// user can use the body to parse to specified struct.
	_ = body
}
