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

	b = []byte{0x01, 00}
	if a, err := Amf0Discovery(b); err != nil {
		t.Error(err)
	} else if err := a.UnmarshalBinary(b); err != nil {
		t.Error(err)
	} else if a, ok := a.(*Amf0Boolean); !ok {
		t.Error("not bool")
	} else if *a != Amf0Boolean(false) {
		t.Error("invalid data")
	}

	b = []byte{0x00, 0x40, 0x59, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if a, err := Amf0Discovery(b); err != nil {
		t.Error(err)
	} else if err := a.UnmarshalBinary(b); err != nil {
		t.Error(err)
	} else if a, ok := a.(*Amf0Number); !ok {
		t.Error("not nubmer")
	} else if *a != Amf0Number(100.0) {
		t.Error("invalid data", *a)
	}

	b = []byte{0x05}
	if a, err := Amf0Discovery(b); err != nil {
		t.Error(err)
	} else if err := a.UnmarshalBinary(b); err != nil {
		t.Error(err)
	} else if _, ok := a.(*Amf0Null); !ok {
		t.Error("not null")
	}

	b = []byte{0x06}
	if a, err := Amf0Discovery(b); err != nil {
		t.Error(err)
	} else if err := a.UnmarshalBinary(b); err != nil {
		t.Error(err)
	} else if _, ok := a.(*Amf0Undefined); !ok {
		t.Error("not undefined")
	}

	b = []byte{0x09}
	if a, err := Amf0Discovery(b); err != nil {
		t.Error(err)
	} else if err := a.UnmarshalBinary(b); err != nil {
		t.Error(err)
	} else if _, ok := a.(*amf0ObjectEOF); !ok {
		t.Error("not object EOF")
	}

	b = []byte{0x0b, 0, 0, 0, 0, 0, 0, 0, 0xf, 0, 0xe}
	if a, err := Amf0Discovery(b); err != nil {
		t.Error(err)
	} else if err := a.UnmarshalBinary(b); err != nil {
		t.Error(err)
	} else if a, ok := a.(*Amf0Date); !ok {
		t.Error("not date")
	} else if a.Date != 0xf || a.Zone != 0xe {
		t.Error("invalid data", *a)
	}

	b = []byte{0x03, 0, 0, 0x09}
	if a, err := Amf0Discovery(b); err != nil {
		t.Error(err)
	} else if err := a.UnmarshalBinary(b); err != nil {
		t.Error(err)
	} else if _, ok := a.(*Amf0Object); !ok {
		t.Error("not object")
	}
}

func TestAmf0Object(t *testing.T) {
	var s Amf0Object
	if err := s.UnmarshalBinary([]byte{0x03, 0, 0, 0x09}); err != nil {
		t.Error("invalid amf0 object")
	}
	if err := s.UnmarshalBinary([]byte{0x03, 0, 2, 'p', 'j', 2, 0, 4, 'o', 'r', 'y', 'x', 0, 0, 9}); err != nil {
		t.Error("invalid amf0 object")
	}
	if p := s.Get("pj").(*Amf0String); p == nil || *p != "oryx" {
		t.Error("invalid amf0 object")
	}

	s = *NewAmf0Object()
	s.Set("pj", NewAmf0String("oryx"))
	if b, err := s.MarshalBinary(); err != nil || len(b) != 15 {
		t.Error("invalid amf0 object")
	}
}

func TestAmf0Date(t *testing.T) {
	var s Amf0Date
	if err := s.UnmarshalBinary([]byte{0x0b, 0, 0, 0, 0, 0, 0, 0, 0xf, 0, 0xe}); err != nil || s.Date != 0xf || s.Zone != 0xe {
		t.Error("invalid amf0 date")
	}

	s = Amf0Date{}
	if b, err := s.MarshalBinary(); err != nil || len(b) != 11 {
		t.Error("invalid amf0 date", b)
	}
}

func TestAmf0Undefined(t *testing.T) {
	var s Amf0Undefined
	if err := s.UnmarshalBinary([]byte{0x06}); err != nil {
		t.Error("invalid amf0 undefined")
	}

	s = Amf0Undefined{}
	if b, err := s.MarshalBinary(); err != nil || len(b) != 1 {
		t.Error("invalid amf0 undefined", b)
	}
}

func TestAmf0Null(t *testing.T) {
	var s Amf0Null
	if err := s.UnmarshalBinary([]byte{0x05}); err != nil {
		t.Error("invalid amf0 null")
	}

	s = Amf0Null{}
	if b, err := s.MarshalBinary(); err != nil || len(b) != 1 {
		t.Error("invalid amf0 null", b)
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

func TestAmf0Boolean(t *testing.T) {
	var s Amf0Boolean
	if err := s.UnmarshalBinary([]byte{0x01, 0x01}); err != nil || !s {
		t.Error("invalid amf0 bool", s)
	}

	s = Amf0Boolean(true)
	if b, err := s.MarshalBinary(); err != nil || len(b) != 2 {
		t.Error("invalid amf0 bool", b)
	}
}

func TestAmf0Number(t *testing.T) {
	var s Amf0Number
	if err := s.UnmarshalBinary([]byte{0x00, 0x40, 0x59, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil || s != 100.0 {
		t.Error("invalid amf0 number")
	}

	s = Amf0Number(100.0)
	if b, err := s.MarshalBinary(); err != nil || len(b) != 9 {
		t.Error("invalid amf0 number", b)
	}
}
