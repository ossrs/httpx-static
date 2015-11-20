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
	"encoding"
	"io"
	"math/rand"
	"runtime/debug"
	"time"
)

// the random object to fill bytes.
var random *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// randome fill the bytes.
func RandomFill(b []byte) {
	for i := 0; i < len(b); i++ {
		// the common value in [0x0f, 0xf0]
		b[i] = byte(0x0f + (random.Int() % (256 - 0x0f - 0x0f)))
	}
}

// invoke the f with recover.
// the name of goroutine, use empty to ignore.
func Recover(name string, f func() error) {
	defer func() {
		if r := recover(); r != nil {
			if name != "" {
				Warn.Println(name, "abort with", r)
			} else {
				Warn.Println("goroutine abort with", r)
			}

			Error.Println(string(debug.Stack()))
		}
	}()

	if err := f(); err != nil && !IsNormalQuit(err) {
		if name != "" {
			Warn.Println(name, "terminated with", err)
		} else {
			Warn.Println("terminated abort with", err)
		}
	}
}

// grow the bytes buffer from reader.
func Grow(in io.Reader, inb *bytes.Buffer, size int) (err error) {
	if inb.Len() >= size {
		return
	}

	if _, err = io.CopyN(inb, in, int64(size)); err != nil {
		return
	}

	return
}

// unmarshaler
type Marshaler interface {
	encoding.BinaryMarshaler
}

// marshal the object o to b
func Marshal(o Marshaler, b *bytes.Buffer) (err error) {
	if b == nil {
		panic("should not be nil.")
	}

	if o == nil {
		panic("should not be nil.")
	}

	if vb, err := o.MarshalBinary(); err != nil {
		return err
	} else if _, err := b.Write(vb); err != nil {
		return err
	}

	return
}

// unmarshaler and sizer.
type UnmarshalSizer interface {
	encoding.BinaryUnmarshaler

	// the total size of bytes for this amf0 instance.
	Size() int
}

// unmarshal the object from b
func Unmarshal(o UnmarshalSizer, b *bytes.Buffer) (err error) {
	if b == nil {
		panic("should not be nil")
	}

	if o == nil {
		panic("should not be nil")
	}

	if err = o.UnmarshalBinary(b.Bytes()); err != nil {
		return
	}
	b.Next(o.Size())

	return
}
