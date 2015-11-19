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
)

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
