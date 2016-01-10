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
	"bufio"
	"bytes"
	"encoding"
	"errors"
	"io"
	"math/rand"
	"reflect"
	"runtime"
	"runtime/debug"
	"time"
)

// the buffered random, for the rand is not thread-safe.
// @see http://stackoverflow.com/questions/14298523/why-does-adding-concurrency-slow-down-this-golang-code
var randoms chan *rand.Rand = make(chan *rand.Rand, runtime.NumCPU())

// randome fill the bytes.
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

// invoke the f with recover.
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

// marshal multiple o, which can be nil.
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

// unmarshal multiple o pointers, which can be nil.
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

// get the first match in flags.
// @return the matched pos in data and the index of flags.
func FirstMatch(data []byte, flags [][]byte) (pos, index int) {
	pos = -1
	index = pos

	for i, flag := range flags {
		if position := bytes.Index(data, flag); position >= 0 {
			if pos > position || pos == -1 {
				pos = position
				index = i
			}
		}
	}

	return
}

// the reader support comment with start and end chars.
type CommentReader struct {
	b *bytes.Buffer
	s *bufio.Scanner
}

var commentNotMatch = errors.New("comment not match")

func NewCommendReader(r io.Reader, startMatches, endMatches [][]byte, isComments, requiredMatches []bool) io.Reader {
	v := &CommentReader{
		s: bufio.NewScanner(r),
		b: &bytes.Buffer{},
	}

	v.s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			// read more.
			return 0, nil, nil
		}

		pos, index := FirstMatch(data, startMatches)
		if pos == -1 || index == -1 {
			if atEOF {
				return len(data), data[:], nil
			}
			return 0, nil, nil
		}

		var extra int
		left := data[pos+len(startMatches[index]):]
		if extra = bytes.Index(left, endMatches[index]); extra == -1 {
			if atEOF {
				if requiredMatches[index] {
					return 0, nil, commentNotMatch
				}
				extra = len(left) - len(endMatches[index])
			} else {
				return 0, nil, nil
			}
		}

		// always consume util end of match.
		advance = pos + len(startMatches[index]) + extra + len(endMatches[index])

		if !isComments[index] {
			return advance, data[:advance], nil
		}
		return advance, data[:pos], nil
	})
	return v
}

// interface io.Reader
func (v *CommentReader) Read(p []byte) (n int, err error) {
	for {
		if v.b.Len() > 0 {
			return v.b.Read(p)
		}

		for v.s.Scan() {
			if len(v.s.Bytes()) > 0 {
				if _, err = v.b.Write(v.s.Bytes()); err != nil {
					return
				}
				break
			}
		}

		if err = v.s.Err(); err != nil {
			return
		}

		if v.b.Len() == 0 {
			return 0, io.EOF
		}
	}

	return
}
