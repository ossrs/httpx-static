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

package core

import (
	"bytes"
	"encoding"
	"math/rand"
	"reflect"
	"runtime"
	"runtime/debug"
	"time"
)

// the buffered random, for the rand is not thread-safe.
// @see http://stackoverflow.com/questions/14298523/why-does-adding-concurrency-slow-down-this-golang-code
var randoms chan *rand.Rand = make(chan *rand.Rand, runtime.NumCPU())

// RandomFill randomly fills the bytes.
func RandomFill(b []byte) {
	// fetch in buffered chan.
	var random *rand.Rand
	select {
	case random = <-randoms:
	default:
		random = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	// use the random.
	for i := 0; i < len(b); i++ {
		// the common value in [0x0f, 0xf0]
		b[i] = byte(0x0f + (random.Int() % (256 - 0x0f - 0x0f)))
	}

	// put back in buffered chan.
	select {
	case randoms <- random:
	default:
	}
}

// Recover invokes the f
// the name of goroutine, use empty to ignore.
func Recover(ctx Context, name string, f func() error) {
	defer func() {
		if r := recover(); r != nil {
			if name != "" {
				Warn.Println(ctx, name, "abort with", r)
			} else {
				Warn.Println(ctx, "goroutine abort with", r)
			}

			Error.Println(ctx, string(debug.Stack()))
		}
	}()

	if err := f(); err != nil && !IsNormalQuit(err) {
		if name != "" {
			Warn.Println(ctx, name, "terminated with", err)
		} else {
			Warn.Println(ctx, "terminated abort with", err)
		}
	}
}

// Marshaler is a marshaler
type Marshaler interface {
	encoding.BinaryMarshaler
}

// Marshal the object o to b
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

// Marshals multiple o, which can be nil.
func Marshals(o ...Marshaler) (data []byte, err error) {
	var b bytes.Buffer

	for _, e := range o {
		if e == nil {
			continue
		}

		if rv := reflect.ValueOf(e); rv.IsNil() {
			continue
		}

		if err = Marshal(e, &b); err != nil {
			return
		}
	}

	return b.Bytes(), nil
}

// UnmarshalSizer is an unmarshaler and sizer.
type UnmarshalSizer interface {
	encoding.BinaryUnmarshaler

	// the total size of bytes for this amf0 instance.
	Size() int
}

// Unmarshal the object from b
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

// Unmarshals multiple o pointers, which can be nil.
func Unmarshals(b *bytes.Buffer, o ...UnmarshalSizer) (err error) {
	for _, e := range o {
		if b.Len() == 0 {
			break
		}

		if e == nil {
			continue
		}

		if rv := reflect.ValueOf(e); rv.IsNil() {
			continue
		}

		if err = e.UnmarshalBinary(b.Bytes()); err != nil {
			return
		}
		b.Next(e.Size())
	}

	return
}
