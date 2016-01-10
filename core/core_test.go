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

package core

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestBytesBuffer(t *testing.T) {
	b := []byte{3, 5, 6}
	if cap(b) != 3 {
		t.Error("invalid")
	}

	bb := bytes.NewBuffer(b)

	bb.Reset()
	bb.WriteByte(5)
	if b[0] != 5 {
		t.Error("invalid")
	}

	bb.WriteByte(6)
	if b[1] != 6 {
		t.Error("invalid")
	}

	bb.WriteByte(7)
	if b[2] != 7 {
		t.Error("invalid")
	}

	bb.WriteByte(8)
	bb.Reset()
	bb.WriteByte(9)
	if b[0] != 5 {
		t.Error("invalid")
	}
}

func TestMain(m *testing.M) {
	Info =  NewLogPlus(log.New(ioutil.Discard, LogInfoLabel, log.LstdFlags))
	Trace = NewLogPlus(log.New(ioutil.Discard, LogTraceLabel, log.LstdFlags))
	Warn =  NewLogPlus(log.New(ioutil.Discard, LogWarnLabel, log.LstdFlags))
	Error = NewLogPlus(log.New(ioutil.Discard, LogErrorLabel, log.LstdFlags))

	os.Exit(m.Run())
}
