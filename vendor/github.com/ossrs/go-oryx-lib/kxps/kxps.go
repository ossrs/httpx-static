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

// The shared objects for kxps.
package kxps

import (
	"fmt"
	ol "github.com/ossrs/go-oryx-lib/logger"
	"sync"
	"time"
)

// The source to stat the requests.
type kxpsSource interface {
	// Get total count.
	Count() uint64
}

// sample for krps.
type sample struct {
	rps        float64
	count      uint64
	create     time.Time
	lastSample time.Time
	// Duration in seconds.
	interval time.Duration
}

func (v *sample) String() string {
	return fmt.Sprintf("rps=%v, count=%v, interval=%v", v.rps, v.count, v.interval)
}

func (v *sample) initialize(now time.Time, nbRequests uint64) {
	v.count = nbRequests
	v.lastSample = now
	v.create = now
}

func (v *sample) sample(now time.Time, nbRequests uint64) bool {
	if v.lastSample.Add(v.interval).After(now) {
		return false
	}

	diff := int64(nbRequests - v.count)
	v.count = nbRequests
	v.lastSample = now
	if diff <= 0 {
		v.rps = 0
		return true
	}

	interval := int(v.interval / time.Millisecond)
	v.rps = float64(diff) * 1000 / float64(interval)

	return true
}

var kxpsClosed = fmt.Errorf("kxps closed")

// The implementation object.
type kxps struct {
	// internal objects.
	source  kxpsSource
	ctx     ol.Context
	closed  bool
	started bool
	lock    *sync.Mutex
	// samples
	r10s  sample
	r30s  sample
	r300s sample
	// for average
	average uint64
	create  time.Time
}

func newKxps(ctx ol.Context, s kxpsSource) *kxps {
	v := &kxps{
		lock:   &sync.Mutex{},
		source: s,
		ctx:    ctx,
	}

	v.r10s.interval = time.Duration(10) * time.Second
	v.r30s.interval = time.Duration(30) * time.Second
	v.r300s.interval = time.Duration(300) * time.Second

	return v
}

func (v *kxps) Close() (err error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	v.closed = true
	v.started = false
	return
}

func (v *kxps) Xps10s() float64 {
	return v.r10s.rps
}

func (v *kxps) Xps30s() float64 {
	return v.r30s.rps
}

func (v *kxps) Xps300s() float64 {
	return v.r300s.rps
}

func (v *kxps) Average() float64 {
	return v.sampleAverage(time.Now())
}

func (v *kxps) sampleAverage(now time.Time) float64 {
	if v.source.Count() == 0 {
		return 0
	}

	if v.average == 0 {
		v.average = v.source.Count()
		v.create = now
		return 0
	}

	diff := int64(v.source.Count() - v.average)
	if diff <= 0 {
		return 0
	}

	duration := int64(now.Sub(v.create) / time.Millisecond)
	if duration <= 0 {
		return 0
	}

	return float64(diff) * 1000 / float64(duration)
}

func (v *kxps) doSample(now time.Time) (err error) {
	count := v.source.Count()
	if count == 0 {
		return
	}

	if v.r10s.count == 0 {
		v.r10s.initialize(now, count)
		v.r30s.initialize(now, count)
		v.r300s.initialize(now, count)
		return
	}

	if !v.r10s.sample(now, count) {
		return
	}

	if !v.r30s.sample(now, count) {
		return
	}

	if !v.r300s.sample(now, count) {
		return
	}

	return
}

func (v *kxps) Start() (err error) {
	ctx := v.ctx

	go func() {
		for {
			if err := v.sample(); err != nil {
				if err == kxpsClosed {
					return
				}
				ol.W(ctx, "kxps ignore sample failed, err is", err)
			}
			time.Sleep(time.Duration(10) * time.Second)
		}
	}()

	v.started = true

	return
}

func (v *kxps) sample() (err error) {
	ctx := v.ctx

	defer func() {
		if r := recover(); r != nil {
			ol.W(ctx, "recover kxps from", r)
		}
	}()

	v.lock.Lock()
	defer v.lock.Unlock()

	if v.closed {
		return kxpsClosed
	}

	return v.doSample(time.Now())
}
