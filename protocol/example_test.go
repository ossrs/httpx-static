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

package protocol_test

import (
	"fmt"
	"github.com/ossrs/go-oryx/protocol"
	"time"
)

func Example_Amf0Discovery() {
	b := []byte{0x02} // read from network

	for len(b) > 0 { // parse all amf0 instance in b.
		var err error
		var a protocol.Amf0Any

		if a, err = protocol.Amf0Discovery(b); err != nil {
			return
		}
		if err = a.UnmarshalBinary(b); err != nil {
			return
		}

		b = b[a.Size():] // consume the bytes for a.

		switch a := a.(type) {
		case *protocol.Amf0String:
			_ = *a // use the *string.
		case *protocol.Amf0Boolean:
			_ = *a // use the *bool.
		case *protocol.Amf0Number:
			_ = *a // use the *float64
		case *protocol.Amf0Null:
			_ = *a // use the null.
		case *protocol.Amf0Undefined:
			_ = *a // use the undefined.
		case *protocol.Amf0Date:
			_ = *a // use the date
		default:
			return // invalid type.
		}
	}
}

func ExampleAmf0String_MarshalBinary() {
	s := protocol.Amf0String("oryx")

	var b []byte
	var err error
	if b, err = s.MarshalBinary(); err != nil {
		return
	}

	fmt.Println(len(b))
	fmt.Println(b)

	// Output:
	// 7
	// [2 0 4 111 114 121 120]
}

func ExampleAmf0String_UnmarshalBinary() {
	b := []byte{0x02, 0x00, 0x04, 'o', 'r', 'y', 'x'} // read from network

	var s protocol.Amf0String
	if err := s.UnmarshalBinary(b); err != nil {
		return
	}

	fmt.Println(s)

	// Output:
	// oryx
}

func ExampleAmf0Boolean_MarshalBinary() {
	s := protocol.Amf0Boolean(true)

	var b []byte
	var err error
	if b, err = s.MarshalBinary(); err != nil {
		return
	}

	fmt.Println(len(b))
	fmt.Println(b)

	// Output:
	// 2
	// [1 1]
}

func ExampleAmf0Boolean_UnmarshalBinary() {
	b := []byte{0x01, 0x01} // read from network

	var s protocol.Amf0Boolean
	if err := s.UnmarshalBinary(b); err != nil {
		return
	}

	fmt.Println(s)

	// Output:
	// true
}

func ExampleAmf0Number_MarshalBinary() {
	s := protocol.Amf0Number(100.0)

	var b []byte
	var err error
	if b, err = s.MarshalBinary(); err != nil {
		return
	}

	fmt.Println(len(b))
	fmt.Println(b)

	// Output:
	// 9
	// [0 64 89 0 0 0 0 0 0]
}

func ExampleAmf0Number_UnmarshalBinary() {
	b := []byte{0x00, 0x40, 0x59, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // read from network

	var s protocol.Amf0Number
	if err := s.UnmarshalBinary(b); err != nil {
		return
	}

	fmt.Println(s)

	// Output:
	// 100
}

func ExampleAmf0Null_MarshalBinary() {
	s := protocol.Amf0Null{}

	var b []byte
	var err error
	if b, err = s.MarshalBinary(); err != nil {
		return
	}

	fmt.Println(len(b))
	fmt.Println(b)

	// Output:
	// 1
	// [5]
}

func ExampleAmf0Null_UnmarshalBinary() {
	b := []byte{0x05} // read from network

	var s protocol.Amf0Null
	if err := s.UnmarshalBinary(b); err != nil {
		return
	}

	fmt.Println("amf0 null")

	// Output:
	// amf0 null
}

func ExampleAmf0Undefined_MarshalBinary() {
	s := protocol.Amf0Undefined{}

	var b []byte
	var err error
	if b, err = s.MarshalBinary(); err != nil {
		return
	}

	fmt.Println(len(b))
	fmt.Println(b)

	// Output:
	// 1
	// [6]
}

func ExampleAmf0Undefined_UnmarshalBinary() {
	b := []byte{0x06} // read from network

	var s protocol.Amf0Undefined
	if err := s.UnmarshalBinary(b); err != nil {
		return
	}

	fmt.Println("amf0 undefined")

	// Output:
	// amf0 undefined
}

func ExampleAmf0Date_MarshalBinary() {
	s := protocol.Amf0Date{}

	s.From(time.Now())

	var b []byte
	var err error
	if b, err = s.MarshalBinary(); err != nil {
		return
	}

	fmt.Println(len(b))
	fmt.Println("amf0 date")

	// Output:
	// 11
	// amf0 date
}

func ExampleAmf0Date_UnmarshalBinary() {
	b := []byte{0x0b, 0, 0, 1, 81, 30, 96, 9, 6, 112, 128} // read from network

	var s protocol.Amf0Date
	if err := s.UnmarshalBinary(b); err != nil {
		return
	}

	fmt.Println("amf0 date")

	// Output:
	// amf0 date
}
