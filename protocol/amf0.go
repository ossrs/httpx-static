// The MIT License (MIT)
//
// Copyright (c) 2013-2016 Oryx(ossrs)
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
	"github.com/ossrs/go-oryx/core"
	"strconv"
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
	core.UnmarshalSizer
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
	case MarkerAmf0EcmaArray:
		return NewAmf0EcmaArray(), nil
	case MarkerAmf0StrictArray:
		return NewAmf0StrictArray(), nil
	case MarkerAmf0Invalid:
		fallthrough
	default:
		return nil, Amf0Error
	}
}

// 2.12 Strict Array Type
// array-count = U32
// strict-array-type = array-count *(value-type)
type Amf0StrictArray struct {
	properties []Amf0Any
}

func NewAmf0StrictArray() *Amf0StrictArray {
	return &Amf0StrictArray{
		properties: make([]Amf0Any, 0),
	}
}

func (v Amf0StrictArray) String() string {
	return fmt.Sprintf("strict-array(%v)", len(v.properties))
}

func (v *Amf0StrictArray) Count() int {
	return int(len(v.properties))
}

func (v *Amf0StrictArray) Get(index int) Amf0Any {
	if index >= len(v.properties) {
		panic("amf0 strict array overflow")
	}
	return v.properties[index]
}

func (v *Amf0StrictArray) Add(e Amf0Any) *Amf0StrictArray {
	v.properties = append(v.properties, e)
	return v
}

func (v *Amf0StrictArray) Size() int {
	var size int = 1 + 4
	for _, e := range v.properties {
		size += e.Size()
	}
	return size
}

func (v *Amf0StrictArray) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0StrictArray); err != nil {
		return
	}

	var count uint32 = uint32(len(v.properties))
	if err = binary.Write(&b, binary.BigEndian, count); err != nil {
		return
	}

	for _, e := range v.properties {
		if err = core.Marshal(e, &b); err != nil {
			return
		}
	}

	return b.Bytes(), nil
}

func (v *Amf0StrictArray) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var m byte
	if m, err = b.ReadByte(); err != nil {
		return
	}

	if m != MarkerAmf0StrictArray {
		return Amf0Error
	}

	var count uint32
	if err = binary.Read(b, binary.BigEndian, &count); err != nil {
		return
	}

	for i := 0; i < int(count); i++ {
		var a Amf0Any
		if a, err = Amf0Discovery(b.Bytes()); err != nil {
			return
		}

		if err = core.Unmarshal(a, b); err != nil {
			return
		}

		v.Add(a)
	}

	return
}

// 2.10 ECMA Array Type
// ecma-array-type = associative-count *(object-property)
// associative-count = U32
// object-property = (UTF-8 value-type) | (UTF-8-empty object-end-marker)
type Amf0EcmaArray struct {
	properties *amf0Properties
}

func NewAmf0EcmaArray() *Amf0EcmaArray {
	return &Amf0EcmaArray{
		properties: NewAmf0Properties(),
	}
}

func (v Amf0EcmaArray) String() string {
	return fmt.Sprintf("ecma-array(%v)", len(v.properties.properties))
}

func (v *Amf0EcmaArray) Set(name string, value Amf0Any) {
	v.properties.Set(name, value)
}

func (v *Amf0EcmaArray) Get(name string) (value Amf0Any) {
	return v.properties.Get(name)
}

func (v *Amf0EcmaArray) Size() int {
	return 1 + 4 + v.properties.Size()
}

func (v *Amf0EcmaArray) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0EcmaArray); err != nil {
		return
	}

	var count uint32
	if err = binary.Write(&b, binary.BigEndian, count); err != nil {
		return
	}

	if err = core.Marshal(v.properties, &b); err != nil {
		return
	}

	return b.Bytes(), nil
}

func (v *Amf0EcmaArray) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var m byte
	if m, err = b.ReadByte(); err != nil {
		return
	}

	if m != MarkerAmf0EcmaArray {
		return Amf0Error
	}

	var count uint32
	if err = binary.Read(b, binary.BigEndian, &count); err != nil {
		return
	}

	if err = core.Unmarshal(v.properties, b); err != nil {
		return
	}

	return
}

// 2.5 Object Type
// anonymous-object-type = object-marker *(object-property)
// object-property = (UTF-8 value-type) | (UTF-8-empty object-end-marker)
type Amf0Object struct {
	properties *amf0Properties
}

func NewAmf0Object() *Amf0Object {
	return &Amf0Object{
		properties: NewAmf0Properties(),
	}
}

func (v Amf0Object) String() string {
	return fmt.Sprintf("object(%v)", len(v.properties.properties))
}

func (v *Amf0Object) Set(name string, value Amf0Any) {
	v.properties.Set(name, value)
}

func (v *Amf0Object) Get(name string) (value Amf0Any) {
	return v.properties.Get(name)
}

func (v *Amf0Object) Size() int {
	return 1 + v.properties.Size()
}

func (v *Amf0Object) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = b.WriteByte(MarkerAmf0Object); err != nil {
		return
	}

	if err = core.Marshal(v.properties, &b); err != nil {
		return
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

	if err = core.Unmarshal(v.properties, b); err != nil {
		return
	}

	return
}

// 2.13 Date Type
// time-zone = S16 ; reserved, not supported should be set to 0x0000
// date-type = date-marker DOUBLE time-zone
// @see: https://github.com/ossrs/srs/issues/185
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

// read amf0 undefined from stream.
// 2.8 undefined Type
// undefined-type = undefined-marker
type Amf0Undefined struct{}

func (v Amf0Undefined) String() string {
	return "undefined"
}

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

// read amf0 null from stream.
// 2.7 null Type
// null-type = null-marker
type Amf0Null struct{}

func (v Amf0Null) String() string {
	return "null"
}

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

// read amf0 number from stream.
// 2.2 Number Type
// number-type = number-marker DOUBLE
// @return default value is 0.
type Amf0Number float64

func NewAmf0Number(v float64) *Amf0Number {
	var n Amf0Number = Amf0Number(v)
	return &n
}

func (v Amf0Number) String() string {
	return strconv.FormatFloat(float64(v), 'f', -1, 64)
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

// read amf0 boolean from stream.
// 2.4 String Type
// boolean-type = boolean-marker U8
//         0 is false, <> 0 is true
// @return default value is false.
type Amf0Boolean bool

func NewAmf0Bool(v bool) *Amf0Boolean {
	var b Amf0Boolean = Amf0Boolean(v)
	return &b
}

func (v Amf0Boolean) String() string {
	if v {
		return "true"
	}
	return "false"
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

// read amf0 string from stream.
// 2.4 String Type
// string-type = string-marker UTF-8
// @return default value is empty string.
// @remark: use SrsAmf0Any::str() to create it.
type Amf0String string

func NewAmf0String(v string) *Amf0String {
	var s Amf0String = Amf0String(v)
	return &s
}

func (v Amf0String) String() string {
	return string(v)
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

// 2.11 Object End Type
// object-end-type = UTF-8-empty object-end-marker
// 0x00 0x00 0x09
// @remark we only use 0x09 as object EOF.
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

// amf0 utf8 string.
// 1.3.1 Strings and UTF-8
// UTF-8 = U16 *(UTF8-char)
// UTF8-char = UTF8-1 | UTF8-2 | UTF8-3 | UTF8-4
// UTF8-1 = %x00-7F
// @remark only support UTF8-1 char.
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
// to ensure in inserted order.
// for the FMLE will crash when AMF0Object is not ordered by inserted,
// if ordered in map, the string compare order, the FMLE will creash when
// get the response of connect app.
type amf0Properties struct {
	properties []*amf0Property
	eof        amf0ObjectEOF
}

func NewAmf0Properties() *amf0Properties {
	return &amf0Properties{
		properties: make([]*amf0Property, 0),
	}
}

func (v *amf0Properties) Set(name string, value Amf0Any) {
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

func (v *amf0Properties) Get(name string) (value Amf0Any) {
	for _, e := range v.properties {
		if string(e.key) == name {
			return e.value
		}
	}
	return
}

func (v *amf0Properties) Size() int {
	var size int = 2 + v.eof.Size()
	for _, e := range v.properties {
		size += e.key.Size()
		size += e.value.Size()
	}
	return size
}

func (v *amf0Properties) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	// properties.
	for _, e := range v.properties {
		if err = core.Marshal(&e.key, &b); err != nil {
			return
		}
		if err = core.Marshal(e.value, &b); err != nil {
			return
		}
	}

	// EOF.
	if _, err = b.Write([]byte{0, 0}); err != nil {
		return
	}

	if err = core.Marshal(&v.eof, &b); err != nil {
		return
	}

	return b.Bytes(), nil
}

func (v *amf0Properties) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	for b.Len() > 0 {
		var key amf0Utf8
		if err = core.Unmarshal(&key, b); err != nil {
			return
		}

		var value Amf0Any
		if value, err = Amf0Discovery(b.Bytes()); err != nil {
			return
		}
		if err = core.Unmarshal(value, b); err != nil {
			return
		}

		// EOF.
		if _, ok := value.(*amf0ObjectEOF); ok && len(key) == 0 {
			break
		}

		v.Set(string(key), value)
	}

	return
}

type amf0Property struct {
	key   amf0Utf8
	value Amf0Any
}
