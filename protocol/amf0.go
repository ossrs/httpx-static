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
	"encoding/binary"
)

// a amf0 string is a string.
type Amf0String string

// encoding.BinaryMarshaler
func (s *Amf0String) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	if err = binary.Write(&b, binary.BigEndian, uint16(len(*s))); err != nil {
		return
	}

	if _, err = b.Write(([]byte)(*s)); err != nil {
		return
	}

	return b.Bytes(), nil
}

// encoding.BinaryUnmarshaler
func (s *Amf0String) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)

	var nb uint16
	if err = binary.Read(b, binary.BigEndian, &nb); err != nil {
		return
	}

	v := make([]byte, nb)
	if _, err = b.Read(v); err != nil {
		return
	}
	*s = Amf0String(string(v))

	return
}
