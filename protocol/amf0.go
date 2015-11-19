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
	"encoding"
	"encoding/binary"
	"fmt"
	"time"
)

// AMF0 marker
const (
	MarkerAmf0Number      = 0x00
	MarkerAmf0Boolean     = 0x01
	MarkerAmf0String      = 0x02
	MarkerAmf0Object      = 0x03
	MarkerAmf0MovieClip   = 0x04 // reserved, not supported
	MarkerAmf0Null        = 0x05
	MarkerAmf0Undefined   = 0x06
	MarkerAmf0Reference   = 0x07
	MarkerAmf0EcmaArray   = 0x08
	MarkerAmf0ObjectEnd   = 0x09
	MarkerAmf0StrictArray = 0x0A
	MarkerAmf0Date        = 0x0B
	MarkerAmf0LongString  = 0x0C
	MarkerAmf0UnSupported = 0x0D
	MarkerAmf0RecordSet   = 0x0E // reserved, not supported
	MarkerAmf0XmlDocument = 0x0F
	MarkerAmf0TypedObject = 0x10
	// AVM+ object is the AMF3 object.
	MarkerAmf0AVMplusObject = 0x11
	// origin array whos data takes the same form as LengthValueBytes
	MarkerAmf0OriginStrictArray = 0x20

	// User defined
	MarkerAmf0Invalid = 0x3F
)

// the amf0 type interface
type Amf0Any interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	// the total size of bytes for this amf0 instance.
	Size() int
}

// discovery the Amf0Any type by marker.
func Amf0Discovery(data []byte) (a Amf0Any, err error) {
	if len(data) == 0 {
		return nil, Amf0Error
	}

	switch data[0] {
	case MarkerAmf0String:
		return NewAmf0String(""), nil
	case MarkerAmf0Boolean:
		return NewAmf0Bool(false), nil
	case MarkerAmf0Number:
		return NewAmf0Number(0), nil
	case MarkerAmf0Null:
		return &Amf0Null{}, nil
	case MarkerAmf0Undefined:
		return &Amf0Undefined{}, nil
	case MarkerAmf0Date:
		return &Amf0Date{}, nil
	case MarkerAmf0ObjectEnd:
		return &amf0ObjectEOF{}, nil
	case MarkerAmf0Object:
		return NewAmf0Object(), nil
	case MarkerAmf0Invalid:
		fallthrough
	default:
		return nil, Amf0Error
	}
}

// an amf0 object is an object.
type Amf0Object struct {
	properties []*amf0Property
	eof        amf0ObjectEOF
}

func NewAmf0Object() *Amf0Object {
	return &Amf0Object{
		properties: make([]*amf0Property, 0),
	}
}

func (v *Amf0Object) Set(name string, value Amf0Any) {
	for _, e := range v.properties {
		if string(e.key) == name {
			e.value = value
			return
		}
	}

	e := &amf0Property{
		key:   amf0Utf8(name),
		value: value,
	}
	v.properties = append(v.properties, e)

	return
}

func (v *Amf0Object) Get(name string) (value Amf0Any) {
	for _, e := range v.properties {
		if string(e.key) == name {
			return e.value
		}
	}
	return
}

func (v *Amf0Object) Size() int {
	var size int = 1 + 2 + v.eof.Size()
	for _, e := range v.properties {
		size += e.key.Size()
		size += e.value.Size()
	}
	return size
}

func (v *Amf0Object) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0Object); err != nil {
		return
	}

	// properties.
	for _, e := range v.properties {
		if vb, err := e.key.MarshalBinary(); err != nil {
			return nil, err
		} else if _, err = b.Write(vb); err != nil {
			return nil, err
		}

		if vb, err := e.value.MarshalBinary(); err != nil {
			return nil, err
		} else if _, err = b.Write(vb); err != nil {
			return nil, err
		}
	}

	// EOF.
	if _, err = b.Write([]byte{0, 0}); err != nil {
		return
	}

	if vb, err := v.eof.MarshalBinary(); err != nil {
		return nil, err
	} else if _, err = b.Write(vb); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (v *Amf0Object) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var m byte
	if m, err = b.ReadByte(); err != nil {
		return
	}

	if m != MarkerAmf0Object {
		return Amf0Error
	}

	for b.Len() > 0 {
		var key amf0Utf8
		if err = key.UnmarshalBinary(b.Bytes()); err != nil {
			return
		}
		b.Next(key.Size())

		var value Amf0Any
		if value, err = Amf0Discovery(b.Bytes()); err != nil {
			return
		}
		if err = value.UnmarshalBinary(b.Bytes()); err != nil {
			return
		}
		b.Next(value.Size())

		// EOF.
		if _, ok := value.(*amf0ObjectEOF); ok && len(key) == 0 {
			break
		}

		v.Set(string(key), value)
	}

	return
}

// an amf0 date is an object.
type Amf0Date struct {
	// date value
	// An ActionScript Date is serialized as the number of milliseconds
	// elapsed since the epoch of midnight on 1st Jan 1970 in the UTC
	// time zone.
	Date uint64
	// time zone
	// While the design of this type reserves room for time zone offset
	// information, it should not be filled in, nor used, as it is unconventional
	// to change time zones when serializing dates on a network. It is suggested
	// that the time zone be queried independently as needed.
	Zone uint16
}

// TODO: parse with system time.Time.
func (v *Amf0Date) From(t time.Time) {
	v.Date = uint64(t.UnixNano() / int64(time.Millisecond))

	_, vz := t.Zone()
	v.Zone = uint16(vz)
}

func (v Amf0Date) String() string {
	return fmt.Sprintf("%v since 1970, zone is %v", v.Date, v.Zone)
}

func (v *Amf0Date) Size() int {
	return 1 + 8 + 2
}

func (v *Amf0Date) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0Date); err != nil {
		return
	}

	if err = binary.Write(&b, binary.BigEndian, v.Date); err != nil {
		return
	}

	if err = binary.Write(&b, binary.BigEndian, v.Zone); err != nil {
		return
	}

	return b.Bytes(), nil
}

func (v *Amf0Date) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var m byte
	if m, err = b.ReadByte(); err != nil {
		return
	}

	if m != MarkerAmf0Date {
		return Amf0Error
	}

	if err = binary.Read(b, binary.BigEndian, &v.Date); err != nil {
		return
	}

	if err = binary.Read(b, binary.BigEndian, &v.Zone); err != nil {
		return
	}

	return
}

// an amf0 undefined is an object.
type Amf0Undefined struct{}

func (v *Amf0Undefined) Size() int {
	return 1
}

func (v *Amf0Undefined) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0Undefined); err != nil {
		return
	}

	return b.Bytes(), nil
}

func (v *Amf0Undefined) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var m byte
	if m, err = b.ReadByte(); err != nil {
		return
	}

	if m != MarkerAmf0Undefined {
		return Amf0Error
	}

	return
}

// an amf0 null is an object.
type Amf0Null struct{}

func (v *Amf0Null) Size() int {
	return 1
}

func (v *Amf0Null) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0Null); err != nil {
		return
	}

	return b.Bytes(), nil
}

func (v *Amf0Null) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var m byte
	if m, err = b.ReadByte(); err != nil {
		return
	}

	if m != MarkerAmf0Null {
		return Amf0Error
	}

	return
}

// an amf0 number is a float64(double)
type Amf0Number float64

func NewAmf0Number(v float64) *Amf0Number {
	var n Amf0Number = Amf0Number(v)
	return &n
}

func (v *Amf0Number) Size() int {
	return 1 + 8
}

func (v *Amf0Number) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0Number); err != nil {
		return
	}

	if err = binary.Write(&b, binary.BigEndian, float64(*v)); err != nil {
		return
	}

	return b.Bytes(), nil
}

func (v *Amf0Number) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var m byte
	if m, err = b.ReadByte(); err != nil {
		return
	}

	if m != MarkerAmf0Number {
		return Amf0Error
	}

	if err = binary.Read(b, binary.BigEndian, (*float64)(v)); err != nil {
		return
	}

	return
}

// an amf0 boolean is a bool.
type Amf0Boolean bool

func NewAmf0Bool(v bool) *Amf0Boolean {
	var b Amf0Boolean = Amf0Boolean(v)
	return &b
}

func (v *Amf0Boolean) Size() int {
	return 1 + 1
}

func (v *Amf0Boolean) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0Boolean); err != nil {
		return
	}

	var vb byte
	if *v {
		vb = 1
	}

	if err = b.WriteByte(vb); err != nil {
		return
	}

	return b.Bytes(), nil
}

func (v *Amf0Boolean) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var m byte
	if m, err = b.ReadByte(); err != nil {
		return
	}

	if m != MarkerAmf0Boolean {
		return Amf0Error
	}

	var vb byte
	if vb, err = b.ReadByte(); err != nil {
		return
	}

	*v = Amf0Boolean(false)
	if vb != 0 {
		*v = Amf0Boolean(true)
	}

	return
}

// an amf0 string is a string.
type Amf0String string

func NewAmf0String(v string) *Amf0String {
	var s Amf0String = Amf0String(v)
	return &s
}

func (v *Amf0String) Size() int {
	return 1 + 2 + len(*v)
}

func (v *Amf0String) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0String); err != nil {
		return
	}

	if err = binary.Write(&b, binary.BigEndian, uint16(len(*v))); err != nil {
		return
	}

	if _, err = b.Write(([]byte)(*v)); err != nil {
		return
	}

	return b.Bytes(), nil
}

func (v *Amf0String) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var m byte
	if m, err = b.ReadByte(); err != nil {
		return
	}

	if m != MarkerAmf0String {
		return Amf0Error
	}

	var nvb uint16
	if err = binary.Read(b, binary.BigEndian, &nvb); err != nil {
		return
	}

	vb := make([]byte, nvb)
	if _, err = b.Read(vb); err != nil {
		return
	}
	*v = Amf0String(string(vb))

	return
}

// an amf0 object EOF is an object.
type amf0ObjectEOF struct{}

func (v *amf0ObjectEOF) Size() int {
	return 1
}

func (v *amf0ObjectEOF) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0ObjectEnd); err != nil {
		return
	}

	return b.Bytes(), nil
}

func (v *amf0ObjectEOF) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var m byte
	if m, err = b.ReadByte(); err != nil {
		return
	}

	if m != MarkerAmf0ObjectEnd {
		return Amf0Error
	}

	return
}

// an amf0 utf8 string is a string.
type amf0Utf8 string

func (s *amf0Utf8) Size() int {
	return 2 + len(*s)
}

func (s *amf0Utf8) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = binary.Write(&b, binary.BigEndian, uint16(len(*s))); err != nil {
		return
	}

	if len(*s) == 0 {
		return
	}

	if _, err = b.Write(([]byte)(*s)); err != nil {
		return
	}

	return b.Bytes(), nil
}

func (s *amf0Utf8) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var nb uint16
	if err = binary.Read(b, binary.BigEndian, &nb); err != nil {
		return
	}

	if nb == 0 {
		return
	}

	v := make([]byte, nb)
	if _, err = b.Read(v); err != nil {
		return
	}
	*s = amf0Utf8(string(v))

	return
}

// the amf0 property for object and array.
type amf0Property struct {
	key   amf0Utf8
	value Amf0Any
}
