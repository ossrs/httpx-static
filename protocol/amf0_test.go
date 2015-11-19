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

import "testing"

func TestAmf0Discovery(t *testing.T) {
	if _, err := Amf0Discovery(nil); err == nil {
		t.Error("invalid")
	}
	if _, err := Amf0Discovery([]byte{}); err == nil {
		t.Error("invalid")
	}

	b := []byte{0x02, 0x00, 0x04, 'o', 'r', 'y', 'x'}
	if a, err := Amf0Discovery(b); err != nil {
		t.Error(err)
	} else if err := a.UnmarshalBinary(b); err != nil {
		t.Error(err)
	} else if a, ok := a.(*Amf0String); !ok {
		t.Error("not string")
	} else if *a != Amf0String("oryx") {
		t.Error("invalid data")
	}
}

func TestAmf0String(t *testing.T) {
	var s Amf0String
	if err := s.UnmarshalBinary([]byte{0x02, 0x00, 0x04, 'o', 'r', 'y', 'x'}); err != nil || len(s) != 4 {
		t.Error("invalid amf0 string", ([]byte)(s))
	}

	s = Amf0String("oryx")
	if b, err := s.MarshalBinary(); err != nil || len(b) != 7 {
		t.Error("invalid amf0 string", b)
	}
}

func TestAmf0Utf8(t *testing.T) {
	var s amf0Utf8
	if err := s.UnmarshalBinary([]byte{0x00, 0x04, 'o', 'r', 'y', 'x'}); err != nil || len(s) != 4 {
		t.Error("invalid amf0 string", ([]byte)(s))
	}

	s = amf0Utf8("oryx")
	if b, err := s.MarshalBinary(); err != nil || len(b) != 6 {
		t.Error("invalid amf0 string", b)
	}
}
