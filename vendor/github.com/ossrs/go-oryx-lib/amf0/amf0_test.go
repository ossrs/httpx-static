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

package amf0

import (
	"bytes"
	"encoding"
	oe "github.com/ossrs/go-oryx-lib/errors"
	"testing"
)

type mockCreateBuffer struct {
	b   buffer
	pob func() buffer
}

func newMockCreateBuffer(b buffer) *mockCreateBuffer {
	v := &mockCreateBuffer{b: b, pob: createBuffer}
	createBuffer = v.create
	return v
}

func (v *mockCreateBuffer) Release() {
	createBuffer = v.pob
}

func (v *mockCreateBuffer) create() buffer {
	return v.b
}

type limitBuffer struct {
	written int
	max     int
}

func newLimitBuffer(max int) buffer {
	return &limitBuffer{max: max}
}

func (v *limitBuffer) Bytes() []byte {
	return nil
}

func (v *limitBuffer) WriteByte(c byte) error {
	v.written++
	if v.written <= v.max {
		return nil
	}
	return oe.New("write byte")
}

func (v *limitBuffer) Write(p []byte) (n int, err error) {
	v.written += len(p)
	if v.written <= v.max {
		return len(p), nil
	}
	return 0, oe.New("write")
}

func TestAmf0Marker(t *testing.T) {
	pvs := []struct {
		m  marker
		ms string
	}{
		{markerNumber, "Number"},
		{markerBoolean, "Boolean"},
		{markerString, "String"},
		{markerObject, "Object"},
		{markerNull, "Null"},
		{markerUndefined, "Undefined"},
		{markerReference, "Reference"},
		{markerEcmaArray, "EcmaArray"},
		{markerObjectEnd, "ObjectEnd"},
		{markerStrictArray, "StrictArray"},
		{markerDate, "Date"},
		{markerLongString, "LongString"},
		{markerUnsupported, "Unsupported"},
		{markerXmlDocument, "XmlDocument"},
		{markerTypedObject, "TypedObject"},
		{markerAvmPlusObject, "AvmPlusObject"},
		{markerMovieClip, "MovieClip"},
		{markerRecordSet, "RecordSet"},
	}
	for _, pv := range pvs {
		if v := pv.m.String(); v != pv.ms {
			t.Errorf("marker %v expect %v actual %v", pv.m, pv.ms, v)
		}
	}
}

func TestDiscovery(t *testing.T) {
	pvs := []struct {
		m  marker
		mv byte
	}{
		{markerNumber, 0},
		{markerBoolean, 1},
		{markerString, 2},
		{markerObject, 3},
		{markerNull, 5},
		{markerUndefined, 6},
		{markerEcmaArray, 8},
		{markerObjectEnd, 9},
		{markerStrictArray, 10},
	}
	for _, pv := range pvs {
		if m, err := Discovery([]byte{pv.mv}); err != nil {
			t.Errorf("discovery err %+v", err)
		} else if v := m.amf0Marker(); v != pv.m {
			t.Errorf("invalid %v expect %v actual %v", pv.mv, pv.m, v)
		}
	}
}

func TestDiscovery2(t *testing.T) {
	pvs := []byte{
		7, 11, 12, 13, 15,
		16, 17, 4,
		14,

		18, 0xff,
	}
	for _, pv := range pvs {
		if m, err := Discovery([]byte{pv}); err == nil {
			t.Errorf("marker=%v should error", pv)
		} else if m != nil {
			t.Errorf("should nil for %v", pv)
		}
	}
}

func TestAmf0UTF8_Size(t *testing.T) {
	if v := amf0UTF8(""); v.Size() != 2 {
		t.Errorf("invalid size %v", v.Size())
	}
	if v := amf0UTF8("oryx"); v.Size() != 2+4 {
		t.Errorf("invalid size %v", v.Size())
	}
}

func TestAmf0UTF8_MarshalBinary(t *testing.T) {
	pvs := []struct {
		b []byte
		v string
	}{
		{[]byte{0, 0}, ""},
		{[]byte{0, 4, 0x6f, 0x72, 0x79, 0x78}, "oryx"},
	}
	for _, pv := range pvs {
		v := amf0UTF8(pv.v)
		if b, err := v.MarshalBinary(); err != nil {
			t.Errorf("marshal %v err %+v", pv.v, err)
		} else if bytes.Compare(b, pv.b) != 0 {
			t.Errorf("invalid data %v expect %v actual %v", pv.v, pv.b, b)
		}
	}
}

func TestAmf0UTF8_UnmarshalBinary(t *testing.T) {
	pvs := []struct {
		b []byte
		v string
	}{
		{[]byte{0, 0}, ""},
		{[]byte{0, 4, 0x6f, 0x72, 0x79, 0x78}, "oryx"},
	}
	for _, pv := range pvs {
		v := amf0UTF8("")
		if err := v.UnmarshalBinary(pv.b); err != nil {
			t.Errorf("unmarshal %v err %+v", pv.b, err)
		} else if string(v) != pv.v {
			t.Errorf("invalid %v expect %v actual %v", pv.b, pv.v, string(v))
		}
	}
}

func TestAmf0UTF8_UnmarshalBinary2(t *testing.T) {
	pvs := [][]byte{
		nil, []byte{}, []byte{0}, []byte{0, 1},
	}
	for _, pv := range pvs {
		v := amf0UTF8("")
		if err := v.UnmarshalBinary(pv); err == nil {
			t.Errorf("should error for %v", pv)
		}
	}
}

func TestAmf0Number_Size(t *testing.T) {
	if v := Number(0); v.Size() != 1+8 {
		t.Errorf("invalid size %v", v.Size())
	}
}

func TestAmf0Number_MarshalBinary(t *testing.T) {
	pvs := []struct {
		b []byte
		v float64
	}{
		{[]byte{0, 0x3f, 0x84, 0x7a, 0xe1, 0x47, 0xae, 0x14, 0x7b}, 0.01},
		{[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0}, 0.0},
	}
	for _, pv := range pvs {
		v := Number(pv.v)
		if b, err := v.MarshalBinary(); err != nil {
			t.Errorf("marshal %v err %+v", pv.v, err)
		} else if bytes.Compare(b, pv.b) != 0 {
			t.Errorf("invalid data %v expect %v actual %v", pv.v, pv.b, b)
		}
	}
}

func TestAmf0Number_UnmarshalBinary(t *testing.T) {
	pvs := []struct {
		b []byte
		v float64
	}{
		{[]byte{0, 0x3f, 0x84, 0x7a, 0xe1, 0x47, 0xae, 0x14, 0x7b}, 0.01},
		{[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0}, 0.0},
	}
	for _, pv := range pvs {
		v := Number(0)
		if err := v.UnmarshalBinary(pv.b); err != nil {
			t.Errorf("unmarshal %v err %+v", pv.b, err)
		} else if float64(v) != pv.v {
			t.Errorf("invalid %v expect %v actual %v", pv.b, pv.v, float64(v))
		}
	}
}

func TestAmf0Number_UnmarshalBinary2(t *testing.T) {
	pvs := [][]byte{
		nil, []byte{}, []byte{0}, []byte{1, 1, 2, 3, 4, 5, 6, 7, 8},
		[]byte{0, 1, 2, 3, 4, 5, 6, 7}, []byte{1},
	}
	for _, pv := range pvs {
		v := Number(0)
		if err := v.UnmarshalBinary(pv); err == nil {
			t.Errorf("should error for %v", pv)
		}
	}
}

func TestAmf0String_Size(t *testing.T) {
	if v := String(""); v.Size() != 1+2 {
		t.Errorf("invalid size %v", v.Size())
	}
	if v := String("oryx"); v.Size() != 1+2+4 {
		t.Errorf("invalid size %v", v.Size())
	}
}

func TestAmf0String_MarshalBinary(t *testing.T) {
	pvs := []struct {
		b []byte
		v string
	}{
		{[]byte{2, 0, 0}, ""},
		{[]byte{2, 0, 4, 0x6f, 0x72, 0x79, 0x78}, "oryx"},
	}
	for _, pv := range pvs {
		v := String(pv.v)
		if b, err := v.MarshalBinary(); err != nil {
			t.Errorf("marshal %v err %+v", pv.v, err)
		} else if bytes.Compare(b, pv.b) != 0 {
			t.Errorf("invalid data %v expect %v actual %v", pv.v, pv.b, b)
		}
	}
}

func TestAmf0String_UnmarshalBinary(t *testing.T) {
	pvs := []struct {
		b []byte
		v string
	}{
		{[]byte{2, 0, 0}, ""},
		{[]byte{2, 0, 4, 0x6f, 0x72, 0x79, 0x78}, "oryx"},
	}
	for _, pv := range pvs {
		v := String("")
		if err := v.UnmarshalBinary(pv.b); err != nil {
			t.Errorf("unmarshal %v err %+v", pv.b, err)
		} else if string(v) != pv.v {
			t.Errorf("invalid %v expect %v actual %v", pv.b, pv.v, string(v))
		}
	}
}

func TestAmf0String_UnmarshalBinary2(t *testing.T) {
	pvs := [][]byte{
		nil, []byte{}, []byte{0, 0},
		[]byte{2, 0},
	}
	for _, pv := range pvs {
		v := String("")
		if err := v.UnmarshalBinary(pv); err == nil {
			t.Errorf("should error for %v", pv)
		}
	}
}

func TestAmf0ObjectEOF_Size(t *testing.T) {
	v := objectEOF{}
	if v.Size() != 3 {
		t.Errorf("invalid size %v", v.Size())
	}
}

func TestAmf0ObjectEOF_MarshalBinary(t *testing.T) {
	v := objectEOF{}
	if b, err := v.MarshalBinary(); err != nil {
		t.Errorf("unmarshal err %+v", err)
	} else if bytes.Compare(b, []byte{0, 0, 9}) != 0 {
		t.Errorf("invalid bytes %v", b)
	}
}

func TestAmf0ObjectEOF_UnmarshalBinary(t *testing.T) {
	v := objectEOF{}
	if err := v.UnmarshalBinary([]byte{0, 0, 9}); err != nil {
		t.Errorf("unmarshal err %+v", err)
	}
}

func TestAmf0ObjectEOF_UnmarshalBinary2(t *testing.T) {
	pvs := [][]byte{
		nil, []byte{}, []byte{0, 0}, []byte{0, 0, 8},
	}
	for _, pv := range pvs {
		v := objectEOF{}
		if err := v.UnmarshalBinary(pv); err == nil {
			t.Errorf("should error for %v", pv)
		}
	}
}

type sizer interface {
	Size() int
}

func sizeof(vs ...sizer) int {
	var size int
	for _, v := range vs {
		size += v.Size()
	}
	return size
}

func TestAmf0ObjectBase_Size(t *testing.T) {
	csk := amf0UTF8("name")
	cs := NewString("oryx")
	cnk := amf0UTF8("years")
	cn := NewNumber(4)
	cbk := amf0UTF8("alive")
	cb := NewBoolean(true)

	pvs := []struct {
		set  func(o *objectBase)
		size int
		err  error
	}{
		{func(o *objectBase) {}, 0, oe.New("empty")},
		{func(o *objectBase) { o.Set(string(csk), cs) }, csk.Size() + cs.Size(), oe.New("one")},
		{func(o *objectBase) {
			o.Set(string(csk), cs).Set(string(cnk), cn)
		}, sizeof(&csk, cs, &cnk, cn), oe.New("two")},
		{func(o *objectBase) {
			o.Set(string(csk), cs).Set(string(cnk), cn).Set(string(cbk), cb)
		}, sizeof(&csk, cs, &cnk, cn, &cbk, cb), oe.New("three")},
	}

	for _, pv := range pvs {
		o := objectBase{}
		pv.set(&o)
		if v := o.Size(); v != pv.size {
			t.Errorf("invalid size %v expect %v err %+v", v, pv.size, pv.err)
		}
	}
}

func concat(vs ...encoding.BinaryMarshaler) []byte {
	b := &bytes.Buffer{}
	for _, v := range vs {
		if vb, err := v.MarshalBinary(); err != nil {
			panic(err)
		} else {
			if _, err = b.Write(vb); err != nil {
				panic(err)
			}
		}
	}
	return b.Bytes()
}

func TestAmf0ObjectBase_MarshalBinary(t *testing.T) {
	csk := amf0UTF8("name")
	cs := NewString("oryx")
	cnk := amf0UTF8("years")
	cn := NewNumber(4)
	cbk := amf0UTF8("alive")
	cb := NewBoolean(true)

	pvs := []struct {
		set func(o *objectBase)
		v   []byte
		err error
	}{
		{func(o *objectBase) {}, []byte{}, oe.New("empty")},
		{func(o *objectBase) { o.Set(string(csk), cs) }, concat(&csk, cs), oe.New("one")},
		{func(o *objectBase) {
			o.Set(string(csk), cs).Set(string(cnk), cn)
		}, concat(&csk, cs, &cnk, cn), oe.New("two")},
		{func(o *objectBase) {
			o.Set(string(csk), cs).Set(string(cnk), cn).Set(string(cbk), cb)
		}, concat(&csk, cs, &cnk, cn, &cbk, cb), oe.New("three")},
	}
	for _, pv := range pvs {
		b := &bytes.Buffer{}
		o := objectBase{}
		pv.set(&o)
		if err := o.marshal(b); err != nil {
			t.Errorf("marshal err %v %+v", err, pv.err)
		} else if bytes.Compare(b.Bytes(), pv.v) != 0 {
			t.Errorf("invalid expect %v actual %v err %+v", pv.v, b.Bytes(), pv.err)
		}
	}
}

func TestAmf0ObjectBase_UnmarshalBinary(t *testing.T) {
	csk := amf0UTF8("name")
	cs := NewString("oryx")
	cnk := amf0UTF8("years")
	cn := NewNumber(4)
	cbk := amf0UTF8("alive")
	cb := NewBoolean(true)
	eof := &objectEOF{}

	pvs := []struct {
		b        []byte
		eof      bool
		maxElems int
		compare  func(o *objectBase) bool
		err      error
	}{
		{concat(eof), true, -1, func(o *objectBase) bool { return true }, oe.New("eof")},
		{concat(&csk, cs, eof), true, -1, func(o *objectBase) bool {
			return o.Get(string(csk)) != nil
		}, oe.New("one")},
		{concat(&csk, cs, &cnk, cn, eof), true, -1, func(o *objectBase) bool {
			return o.Get(string(csk)) != nil && o.Get(string(cnk)) != nil
		}, oe.New("two")},
		{concat(&csk, cs, &cnk, cn, &cbk, cb, eof), true, -1, func(o *objectBase) bool {
			return o.Get(string(csk)) != nil && o.Get(string(cnk)) != nil && o.Get(string(cbk)) != nil
		}, oe.New("two")},
		{[]byte{}, false, 0, func(o *objectBase) bool { return true }, oe.New("empty")},
		{concat(&csk, cs), false, 1, func(o *objectBase) bool {
			return o.Get(string(csk)) != nil
		}, oe.New("one")},
		{concat(&csk, cs, &cnk, cn), false, 2, func(o *objectBase) bool {
			return o.Get(string(csk)) != nil && o.Get(string(cnk)) != nil
		}, oe.New("two")},
		{concat(&csk, cs, &cnk, cn, &cbk, cb), false, 3, func(o *objectBase) bool {
			return o.Get(string(csk)) != nil && o.Get(string(cnk)) != nil && o.Get(string(cbk)) != nil
		}, oe.New("two")},
	}
	for _, pv := range pvs {
		v := &objectBase{}
		if err := v.unmarshal(pv.b, pv.eof, pv.maxElems); err != nil {
			t.Errorf("unmarshal %v err %v %+v", pv.b, err, pv.err)
		} else if !pv.compare(v) {
			t.Errorf("invalid object err %+v", pv.err)
		}
	}
}

func TestAmf0ObjectBase_UnmarshalBinary2(t *testing.T) {
	pvs := []struct {
		b        []byte
		eof      bool
		maxElems int
	}{
		{[]byte{}, true, 1},
		{[]byte{0}, true, -1},
		{[]byte{0, 0}, true, -1},
		{[]byte{0, 0, 0}, true, -1},
		{[]byte{}, false, -1},
		{[]byte{0, 0, 0}, false, -1},
		{[]byte{0, 1, byte('e'), 0}, false, 1},
		{[]byte{0, 1, byte('e'), 0}, false, 2},
	}
	for _, pv := range pvs {
		v := &objectBase{}
		if err := v.unmarshal(pv.b, pv.eof, pv.maxElems); err == nil {
			t.Errorf("should error for %v", pv)
		}
	}
}

func TestAmf0ObjectBase_Marshal2(t *testing.T) {
	csk := amf0UTF8("name")
	cs := NewString("oryx")

	pvs := []struct {
		mb  func() *mockCreateBuffer
		set func(o *objectBase)
		err error
	}{
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(0))
		}, func(o *objectBase) { o.Set(string(csk), cs) }, oe.New("zero")},
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(csk.Size()))
		}, func(o *objectBase) { o.Set(string(csk), cs) }, oe.New("one")},
	}

	for _, pv := range pvs {
		func() {
			m := pv.mb()
			defer m.Release()

			o := &objectBase{}
			pv.set(o)

			if err := o.marshal(m.b); err == nil {
				t.Errorf("should error %+v", pv.err)
			}
		}()
	}
}

func TestAmf0ObjectBase_Get(t *testing.T) {
	o := &objectBase{}
	if v := o.Get("key"); v != nil {
		t.Errorf("should nil, actual %v", v)
	}
}

func TestAmf0ObjectBase_Set(t *testing.T) {
	o := &objectBase{}
	o.Set("key", NewString("name")).Set("key", NewString("age"))
	if v := o.Get("key"); v == nil {
		t.Error("invalid key")
	} else if v, ok := v.(*String); !ok {
		t.Error("invalid key")
	} else if string(*v) != "age" {
		t.Errorf("invalid value %v", string(*v))
	}
}

func TestAmf0Object_Size(t *testing.T) {
	if v := NewObject(); v.Size() != 1+3 {
		t.Errorf("invalid size %v", v.Size())
	}
}

func TestAmf0Object_MarshalBinary(t *testing.T) {
	v := NewObject()
	if b, err := v.MarshalBinary(); err != nil {
		t.Errorf("marshal failed err %+v", err)
	} else if bytes.Compare(b, []byte{3, 0, 0, 9}) != 0 {
		t.Errorf("invalid object %v", b)
	}
}

func TestAmf0Object_UnmarshalBinary(t *testing.T) {
	b := []byte{3, 0, 0, 9}
	v := NewObject()
	if err := v.UnmarshalBinary(b); err != nil {
		t.Errorf("unmarshal failed err %+v", err)
	}
}

func TestAmf0Object_UnmarshalBinary2(t *testing.T) {
	pvs := [][]byte{
		nil, []byte{}, []byte{0},
		[]byte{3, 0},
	}
	for _, pv := range pvs {
		v := NewObject()
		if err := v.UnmarshalBinary(pv); err == nil {
			t.Errorf("should error for %v", pv)
		}
	}
}

func TestAmf0Object_MarshalBinary2(t *testing.T) {
	csk := amf0UTF8("name")
	cs := NewString("oryx")

	pvs := []struct {
		mb  func() *mockCreateBuffer
		set func(o *Object)
		err error
	}{
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(0))
		}, func(o *Object) {}, oe.New("zero")},
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(1))
		}, func(o *Object) { o.Set(string(csk), cs) }, oe.New("zero")},
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(1))
		}, func(o *Object) {}, oe.New("one")},
	}

	for _, pv := range pvs {
		func() {
			m := pv.mb()
			defer m.Release()

			o := NewObject()
			pv.set(o)

			if _, err := o.MarshalBinary(); err == nil {
				t.Errorf("should error %+v", pv.err)
			}
		}()
	}
}

func TestAmf0EcmaArray_Size(t *testing.T) {
	if v := NewEcmaArray(); v.Size() != 1+4+3 {
		t.Errorf("invalid size %v", v.Size())
	}
}

func TestAmf0EcmaArray_MarshalBinary(t *testing.T) {
	v := NewEcmaArray()
	if b, err := v.MarshalBinary(); err != nil {
		t.Errorf("marshal failed err %+v", err)
	} else if bytes.Compare(b, []byte{8, 0, 0, 0, 0, 0, 0, 9}) != 0 {
		t.Errorf("invalid object %v", b)
	}
}

func TestAmf0EcmaArray_UnmarshalBinary(t *testing.T) {
	b := []byte{8, 0, 0, 0, 0, 0, 0, 9}
	v := NewEcmaArray()
	if err := v.UnmarshalBinary(b); err != nil {
		t.Errorf("unmarshal failed err %+v", err)
	}
}

func TestAmf0EcmaArray_UnmarshalBinary2(t *testing.T) {
	pvs := [][]byte{
		nil, []byte{}, []byte{0},
		[]byte{8, 0, 0, 0}, []byte{8, 0, 0, 0, 0, 0},
		[]byte{0, 0, 0, 0, 0, 0},
	}
	for _, pv := range pvs {
		v := NewEcmaArray()
		if err := v.UnmarshalBinary(pv); err == nil {
			t.Errorf("should error for %v", pv)
		}
	}
}

func TestAmf0EcmaArray_MarshalBinary2(t *testing.T) {
	csk := amf0UTF8("name")
	cs := NewString("oryx")

	pvs := []struct {
		mb  func() *mockCreateBuffer
		set func(o *EcmaArray)
		err error
	}{
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(0))
		}, func(o *EcmaArray) {}, oe.New("zero")},
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(4))
		}, func(o *EcmaArray) {}, oe.New("zero")},
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(5))
		}, func(o *EcmaArray) { o.Set(string(csk), cs) }, oe.New("zero")},
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(5))
		}, func(o *EcmaArray) {}, oe.New("one")},
	}

	for _, pv := range pvs {
		func() {
			m := pv.mb()
			defer m.Release()

			o := NewEcmaArray()
			pv.set(o)

			if _, err := o.MarshalBinary(); err == nil {
				t.Errorf("should error %+v", pv.err)
			}
		}()
	}
}

func TestAmf0StrictArray_Size(t *testing.T) {
	if v := NewStrictArray(); v.Size() != 1+4 {
		t.Errorf("invalid size %v", v.Size())
	}
}

func TestAmf0StrictArray_MarshalBinary(t *testing.T) {
	v := NewStrictArray()
	if b, err := v.MarshalBinary(); err != nil {
		t.Errorf("marshal failed err %+v", err)
	} else if bytes.Compare(b, []byte{10, 0, 0, 0, 0}) != 0 {
		t.Errorf("invalid object %v", b)
	}
}

func TestAmf0StrictArray_UnmarshalBinary(t *testing.T) {
	pvs := [][]byte{
		[]byte{10, 0, 0, 0, 0},
		[]byte{10, 0, 0, 0, 1, 0, 1, byte('e'), 5},
	}

	for _, pv := range pvs {
		v := NewStrictArray()
		if err := v.UnmarshalBinary(pv); err != nil {
			t.Errorf("unmarshal failed err %+v", err)
		}
	}
}

func TestAmf0StrictArray_UnmarshalBinary2(t *testing.T) {
	pvs := [][]byte{
		nil, []byte{}, []byte{0},
		[]byte{10, 0, 0, 0}, []byte{0, 0, 0, 0, 0},
		[]byte{10, 0, 0, 0, 1},
	}
	for _, pv := range pvs {
		v := NewStrictArray()
		if err := v.UnmarshalBinary(pv); err == nil {
			t.Errorf("should error for %v", pv)
		}
	}
}

func TestAmf0StrictArray_MarshalBinary2(t *testing.T) {
	csk := amf0UTF8("name")
	cs := NewString("oryx")

	pvs := []struct {
		mb  func() *mockCreateBuffer
		set func(o *StrictArray)
		err error
	}{
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(0))
		}, func(o *StrictArray) {}, oe.New("zero")},
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(4))
		}, func(o *StrictArray) {}, oe.New("zero")},
		{func() *mockCreateBuffer {
			return newMockCreateBuffer(newLimitBuffer(5))
		}, func(o *StrictArray) { o.Set(string(csk), cs) }, oe.New("zero")},
	}

	for _, pv := range pvs {
		func() {
			m := pv.mb()
			defer m.Release()

			o := NewStrictArray()
			pv.set(o)

			if _, err := o.MarshalBinary(); err == nil {
				t.Errorf("should error %+v", pv.err)
			}
		}()
	}
}

func TestAmf0SingleMarker_Size(t *testing.T) {
	v := newSingleMarkerObject(markerBoolean)
	if v.Size() != 1 {
		t.Errorf("invalid size %v", v.Size())
	}
}

func TestAmf0SingleMarker_MarshalBinary(t *testing.T) {
	v := newSingleMarkerObject(markerBoolean)
	if b, err := v.MarshalBinary(); err != nil {
		t.Errorf("marshal failed err %+v", err)
	} else if bytes.Compare(b, []byte{1}) != 0 {
		t.Errorf("invalid object %v", b)
	}
}

func TestAmf0SingleMarker_UnmarshalBinary(t *testing.T) {
	b := []byte{1}
	v := newSingleMarkerObject(markerBoolean)
	if err := v.UnmarshalBinary(b); err != nil {
		t.Errorf("unmarshal failed err %+v", err)
	} else if v.target != markerBoolean {
		t.Errorf("invalid target %v", v.target)
	}
}

func TestAmf0SingleMarker_UnmarshalBinary2(t *testing.T) {
	pvs := [][]byte{
		nil, []byte{}, []byte{0},
	}
	for _, pv := range pvs {
		v := newSingleMarkerObject(markerBoolean)
		if err := v.UnmarshalBinary(pv); err == nil {
			t.Errorf("should error for %v", pv)
		}
	}
}

func TestAmf0Boolean_Size(t *testing.T) {
	v := NewBoolean(true)
	if v.Size() != 2 {
		t.Errorf("invalid size %v", v.Size())
	}
}

func TestAmf0Boolean_MarshalBinary(t *testing.T) {
	v := NewBoolean(true)
	if b, err := v.MarshalBinary(); err != nil {
		t.Errorf("marshal failed err %+v", err)
	} else if bytes.Compare(b, []byte{1, 1}) != 0 {
		t.Errorf("invalid object %v", b)
	}
}

func TestAmf0Boolean_UnmarshalBinary(t *testing.T) {
	b := []byte{1, 0}
	v := NewBoolean(true)
	if err := v.UnmarshalBinary(b); err != nil {
		t.Errorf("unmarshal failed err %+v", err)
	} else if v, ok := v.(*Boolean); !ok {
		t.Errorf("invalid type")
	} else if bool(*v) != false {
		t.Errorf("invalid target %v", *v)
	}
}

func TestAmf0Boolean_UnmarshalBinary2(t *testing.T) {
	pvs := [][]byte{
		nil, []byte{}, []byte{1},
		[]byte{0, 0},
	}
	for _, pv := range pvs {
		v := NewBoolean(true)
		if err := v.UnmarshalBinary(pv); err == nil {
			t.Errorf("should error for %v", pv)
		}
	}
}
