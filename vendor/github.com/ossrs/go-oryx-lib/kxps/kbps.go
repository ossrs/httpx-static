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

// The kbps is about the bitrate.
package kxps

import (
	ol "github.com/ossrs/go-oryx-lib/logger"
	"io"
)

// The source to stat the bitrate.
type KbpsSource interface {
	// Get total number of bytes.
	TotalBytes() uint64
}

// The object to calc the kbps.
type Kbps interface {
	// Start the kbps sample goroutine.
	Start() (err error)

	// Get the kbps in last 10s.
	Kbps10s() float64
	// Get the kbps in last 30s.
	Kbps30s() float64
	// Get the kbps in last 300s.
	Kbps300s() float64
	// Get the kbps in average
	Average() float64

	// When closed, this kbps should never use again.
	io.Closer
}

type kbps struct {
	source KbpsSource
	imp    *kxps
}

func NewKbps(ctx ol.Context, source KbpsSource) Kbps {
	v := &kbps{source: source}
	v.imp = newKxps(ctx, v)
	return v
}

func (v *kbps) Count() uint64 {
	return v.source.TotalBytes()
}

func (v *kbps) Close() (err error) {
	return v.imp.Close()
}

func (v *kbps) Kbps10s() float64 {
	if !v.imp.started {
		panic("should start kbps first.")
	}
	// Bps to Kbps
	return v.imp.Xps10s() * 8 / 1000
}

func (v *kbps) Kbps30s() float64 {
	if !v.imp.started {
		panic("should start kbps first.")
	}
	// Bps to Kbps
	return v.imp.Xps30s() * 8 / 1000
}

func (v *kbps) Kbps300s() float64 {
	if !v.imp.started {
		panic("should start kbps first.")
	}
	// Bps to Kbps
	return v.imp.Xps300s() * 8 / 1000
}

func (v *kbps) Average() float64 {
	if !v.imp.started {
		panic("should start kbps first.")
	}
	// Bps to Kbps
	return v.imp.Average() * 8 / 1000
}

func (v *kbps) Start() (err error) {
	return v.imp.Start()
}
