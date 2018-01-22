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

package kxps_test

import (
	"github.com/ossrs/go-oryx-lib/kxps"
)

func ExampleKrps() {
	// user must provides the krps source
	var source kxps.KrpsSource

	krps := kxps.NewKrps(nil, source)
	defer krps.Close()

	if err := krps.Start(); err != nil {
		return
	}

	_ = krps.Average()
	_ = krps.Rps10s()
	_ = krps.Rps30s()
	_ = krps.Rps300s()
}

func ExampleKbps() {
	// user must provides the kbps source
	var source kxps.KbpsSource

	kbps := kxps.NewKbps(nil, source)
	defer kbps.Close()

	if err := kbps.Start(); err != nil {
		return
	}

	_ = kbps.Average()
	_ = kbps.Kbps10s()
	_ = kbps.Kbps30s()
	_ = kbps.Kbps300s()
}
