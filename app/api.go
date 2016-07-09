// The MIT License (MIT)
//
// Copyright (c) 2013-2016 Oryx(ossrs)
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

package app

import (
	"os"
	"time"

	"github.com/ossrs/go-oryx/core"
)

type Summary struct {
	Ok   bool  `json:"ok"`
	Now  int64 `json:"now_ms"`
	Self struct {
		Version string `json:"version"`
		Pid     int64  `json:"pid"`
		Ppid    int64  `json:"ppid"`
	} `json:"self"`
}

func NewSummary() *Summary {
	s := &Summary{}

	s.Now = time.Now().UnixNano() / int64(time.Millisecond)

	s.Self.Version = core.Version()
	s.Self.Pid = int64(os.Getpid())
	s.Self.Ppid = int64(os.Getppid())

	return s
}
