// The MIT License (MIT)
//
// Copyright (c) 2013-2015 Oryx(ossrs)
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

// the HLS+ research is for streaming HLS solution,
// to finger out use sub m3u8 or 302 redirect to generate id,
// and how to generate the stream.
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
)

type M3u8Writer struct {
	dir  http.Dir
	uuid string
}

func (v *M3u8Writer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	var err error
	var f http.File
	if f, err = v.dir.Open(path); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer f.Close()

	var b []byte
	if b, err = ioutil.ReadAll(f); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	//fmt.Println("m3u8 writer process", len(b))
	p := strings.Replace(string(b), ".ts", ".ts?uuid="+v.uuid, -1)
	if _, err = w.Write([]byte(p)); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
}

func main() {
	fmt.Println("HLS+ research.")

	addr := fmt.Sprintf("0.0.0.0:8080")
	dir := http.Dir("./")
	fmt.Println("serve", addr, "at", dir)

	fh := http.FileServer(dir)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ext := path.Ext(r.URL.Path)
		if ext != ".m3u8" && ext != ".ts" {
			fh.ServeHTTP(w, r)
			return
		}

		fmt.Println("HLS+ serve", r.URL.String())
		q := r.URL.Query()
		uuid := q.Get("uuid")

		if ext == ".m3u8" {
			if len(uuid) == 0 {
				q.Set("uuid", "winlin")
				to := fmt.Sprintf("%v?%v", r.URL.Path, q.Encode())
				fmt.Println("redirect", ext, r.URL.String(), "to", to)

				http.Redirect(w, r, to, http.StatusFound)
			} else {
				mw := &M3u8Writer{dir: dir, uuid: uuid}
				mw.ServeHTTP(w, r)
			}
			return
		}

		if ext == ".ts" {
			if len(uuid) == 0 {
				http.Error(w, "Directly TS not allowed", http.StatusUnauthorized)
			} else {
				fh.ServeHTTP(w, r)
			}
			return
		}

		http.NotFound(w, r)
	})
	if err := http.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}
}
