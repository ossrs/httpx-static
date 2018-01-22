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

package kxps

import (
	"testing"
	"time"
)

type mockSource struct {
	s uint64
}

func (v *mockSource) Count() uint64 {
	return v.s
}

func TestKxps_Average(t *testing.T) {
	s := &mockSource{}
	kxps := newKxps(nil, s)

	if v := kxps.sampleAverage(time.Unix(0, 0)); v != 0 {
		t.Errorf("invalid average %v", v)
	}

	s.s = 10
	if v := kxps.sampleAverage(time.Unix(10, 0)); v != 0 {
		t.Errorf("invalid average %v", v)
	}

	s.s = 20
	if v := kxps.sampleAverage(time.Unix(10, 0)); v != 0 {
		t.Errorf("invalid average %v", v)
	} else if v := kxps.sampleAverage(time.Unix(20, 0)); v != 10.0/10.0 {
		t.Errorf("invalid average %v", v)
	}
}

func TestKxps_Rps10s(t *testing.T) {
	s := &mockSource{}
	kxps := newKxps(nil, s)

	if err := kxps.doSample(time.Unix(0, 0)); err != nil {
		t.Errorf("sample failed, err is", err)
	} else if kxps.Xps10s() != 0 || kxps.Xps30s() != 0 || kxps.Xps300s() != 0 {
		t.Errorf("sample invalid, 10s=%v, 30s=%v, 300s=%v", kxps.Xps10s(), kxps.Xps30s(), kxps.Xps300s())
	}

	s.s = 10
	if err := kxps.doSample(time.Unix(10, 0)); err != nil {
		t.Errorf("sample failed, err is", err)
	} else if kxps.Xps10s() != 0 || kxps.Xps30s() != 0 || kxps.Xps300s() != 0 {
		t.Errorf("sample invalid, 10s=%v, 30s=%v, 300s=%v", kxps.Xps10s(), kxps.Xps30s(), kxps.Xps300s())
	}

	s.s = 20
	if err := kxps.doSample(time.Unix(20, 0)); err != nil {
		t.Errorf("sample failed, err is", err)
	} else if kxps.Xps10s() != 10.0/10.0 || kxps.Xps30s() != 0 || kxps.Xps300s() != 0 {
		t.Errorf("sample invalid, 10s=%v, 30s=%v, 300s=%v", kxps.Xps10s(), kxps.Xps30s(), kxps.Xps300s())
	} else if err := kxps.doSample(time.Unix(30, 0)); err != nil {
		t.Errorf("sample failed, err is", err)
	} else if kxps.Xps10s() != 0 || kxps.Xps30s() != 0 || kxps.Xps300s() != 0 {
		t.Errorf("sample invalid, 10s=%v, 30s=%v, 300s=%v", kxps.Xps10s(), kxps.Xps30s(), kxps.Xps300s())
	}

	s.s = 30
	if err := kxps.doSample(time.Unix(40, 0)); err != nil {
		t.Errorf("sample failed, err is", err)
	} else if kxps.Xps10s() != 10.0/10.0 || kxps.Xps30s() != 20.0/30.0 || kxps.Xps300s() != 0 {
		t.Errorf("sample invalid, 10s=%v, 30s=%v, 300s=%v", kxps.Xps10s(), kxps.Xps30s(), kxps.Xps300s())
	} else if err := kxps.doSample(time.Unix(50, 0)); err != nil {
		t.Errorf("sample failed, err is", err)
	} else if kxps.Xps10s() != 0 || kxps.Xps30s() != 20.0/30.0 || kxps.Xps300s() != 0 {
		t.Errorf("sample invalid, 10s=%v, 30s=%v, 300s=%v", kxps.Xps10s(), kxps.Xps30s(), kxps.Xps300s())
	}

	s.s = 40
	if err := kxps.doSample(time.Unix(310, 0)); err != nil {
		t.Errorf("sample failed, err is", err)
	} else if kxps.Xps10s() != 10.0/10.0 || kxps.Xps30s() != 10.0/30.0 || kxps.Xps300s() != 30.0/300.0 {
		t.Errorf("sample invalid, 10s=%v, 30s=%v, 300s=%v", kxps.Xps10s(), kxps.Xps30s(), kxps.Xps300s())
	} else if err := kxps.doSample(time.Unix(320, 0)); err != nil {
		t.Errorf("sample failed, err is", err)
	} else if kxps.Xps10s() != 0 || kxps.Xps30s() != 10.0/30.0 || kxps.Xps300s() != 30.0/300.0 {
		t.Errorf("sample invalid, 10s=%v, 30s=%v, 300s=%v", kxps.Xps10s(), kxps.Xps30s(), kxps.Xps300s())
	} else if err := kxps.doSample(time.Unix(340, 0)); err != nil {
		t.Errorf("sample failed, err is", err)
	} else if kxps.Xps10s() != 0 || kxps.Xps30s() != 0 || kxps.Xps300s() != 30.0/300.0 {
		t.Errorf("sample invalid, 10s=%v, 30s=%v, 300s=%v", kxps.Xps10s(), kxps.Xps30s(), kxps.Xps300s())
	} else if err := kxps.doSample(time.Unix(610, 0)); err != nil {
		t.Errorf("sample failed, err is", err)
	} else if kxps.Xps10s() != 0 || kxps.Xps30s() != 0 || kxps.Xps300s() != 0 {
		t.Errorf("sample invalid, 10s=%v, 30s=%v, 300s=%v", kxps.Xps10s(), kxps.Xps30s(), kxps.Xps300s())
	}
}
