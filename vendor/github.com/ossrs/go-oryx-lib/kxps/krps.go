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

// The krps is about the request or message rate.
package kxps

import (
	ol "github.com/ossrs/go-oryx-lib/logger"
	"io"
)

// The source to stat the requests.
type KrpsSource interface {
	// Get total number of requests.
	NbRequests() uint64
}

// The object to calc the krps.
type Krps interface {
	// Start the krps sample goroutine.
	Start() (err error)

	// Get the rps in last 10s.
	Rps10s() float64
	// Get the rps in last 30s.
	Rps30s() float64
	// Get the rps in last 300s.
	Rps300s() float64
	// Get the rps in average
	Average() float64

	// When closed, this krps should never use again.
	io.Closer
}

// The implementation object.
type krps struct {
	source KrpsSource
	imp    *kxps
}

func NewKrps(ctx ol.Context, s KrpsSource) Krps {
	v := &krps{
		source: s,
	}
	v.imp = newKxps(ctx, v)
	return v
}

func (v *krps) Count() uint64 {
	return v.source.NbRequests()
}

func (v *krps) Close() (err error) {
	return v.imp.Close()
}

func (v *krps) Rps10s() float64 {
	if !v.imp.started {
		panic("should start krps first.")
	}
	return v.imp.Xps10s()
}

func (v *krps) Rps30s() float64 {
	if !v.imp.started {
		panic("should start krps first.")
	}
	return v.imp.Xps30s()
}

func (v *krps) Rps300s() float64 {
	if !v.imp.started {
		panic("should start krps first.")
	}
	return v.imp.Xps300s()
}

func (v *krps) Average() float64 {
	if !v.imp.started {
		panic("should start krps first.")
	}
	return v.imp.Average()
}

func (v *krps) Start() (err error) {
	return v.imp.Start()
}
