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

package protocol

import (
	"bytes"
	"testing"
)

func TestBytesWriter(t *testing.T) {
	w := NewBytesWriter(make([]byte, 1))
	if n, err := w.Write(make([]byte, 1)); err != nil || n != 1 {
		t.Error("should be ok")
	}
	if n, err := w.Write(make([]byte, 0)); err != nil || n != 0 {
		t.Error("should be ok")
	}
	if n, err := w.Write(nil); err != nil || n != 0 {
		t.Error("should be ok")
	}
	if n, err := w.Write(make([]byte, 2)); err == nil || n != 0 {
		t.Error("should not be ok")
	}

	w = NewBytesWriter(make([]byte, 10))
	if n, err := w.Write(make([]byte, 5)); err != nil || n != 5 {
		t.Error("should be ok")
	}
	if n, err := w.Write(make([]byte, 5)); err != nil || n != 5 {
		t.Error("should be ok")
	}
	if n, err := w.Write(make([]byte, 1)); err == nil || n != 0 {
		t.Error("should not be ok")
	}
}

func TestHsBytes(t *testing.T) {
	b := NewHsBytes()
	if len(b.c0c1c2) != 3073 {
		t.Error("c0c1c2 should be 3073B")
	}
	if len(b.s0s1s2) != 3073 {
		t.Error("s0s1s2 should be 3073B")
	}
	if len(b.C0()) != 1 {
		t.Error("c0 should be 1B")
	}
	if len(b.C1()) != 1536 {
		t.Error("c1 should be 1536B")
	}
	if len(b.C2()) != 1536 {
		t.Error("c2 should be 1536B")
	}
	if len(b.S0()) != 1 {
		t.Error("s0 should be 1B")
	}
	if len(b.S1()) != 1536 {
		t.Error("s1 should be 1536B")
	}
	if len(b.S2()) != 1536 {
		t.Error("s2 should be 1536B")
	}
}

func TestHsBytes_Plaintext(t *testing.T) {
	b := NewHsBytes()

	b.C0()[0] = 0x03
	if !b.ClientPlaintext() {
		t.Error("should be plaintext")
	}

	b.C0()[0] = 0x04
	if b.ClientPlaintext() {
		t.Error("should not be plaintext")
	}

	b.S0()[0] = 0x03
	if !b.ServerPlaintext() {
		t.Error("should be plaintext")
	}
	b.S0()[0] = 0x04
	if b.ServerPlaintext() {
		t.Error("should not be plaintext")
	}
}

func TestHsBytes_readC0C1(t *testing.T) {
	b := NewHsBytes()

	d := make([]byte, 1537)
	d[0] = 0x0f
	d[1536] = 0x0f
	if err := b.readC0C1(bytes.NewReader(d)); err != nil || !b.c0c1Ok {
		t.Error("should be ok")
	}
	if b.C0()[0] != 0x0f || b.C1()[1535] != 0x0f {
		t.Error("invalid value")
	}

	d = make([]byte, 1536)
	d[0] = 0x0f
	d[1535] = 0x0f
	if err := b.readC2(bytes.NewReader(d)); err != nil || !b.c2Ok {
		t.Error("should be ok")
	}
	if b.C2()[0] != 0x0f || b.C2()[1535] != 0x0f {
		t.Error("invalid value")
	}
}

func TestHsBytes_createS0S1S2(t *testing.T) {
	b := NewHsBytes()

	d := make([]byte, 1537)
	d[0] = 0x0f
	d[1] = 0x0e
	d[2] = 0x0d
	d[3] = 0x0c
	d[4] = 0x0b
	d[1536] = 0x0f
	if err := b.readC0C1(bytes.NewReader(d)); err != nil {
		t.Error("should be ok")
	}

	b.createS0S1S2()
	if b.S0()[0] != 0x03 {
		t.Error("should be plaintext")
	}
	if !bytes.Equal(b.s1Time2(), b.c1Time()) {
		t.Error("invalid time")
	}
	if !bytes.Equal(b.C1(), b.S2()) {
		t.Error("invalid time")
	}
}
