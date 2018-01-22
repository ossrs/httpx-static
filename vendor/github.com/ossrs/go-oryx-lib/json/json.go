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

// The oryx json package support json with c++ style comments.
// User can use the following APIs:
//		Unmarshal, directly unmarshal a Reader to object, like json.Unmarshal
//		NewJsonPlusReader, convert the Reader to data stream without comments.
//		NewCommentReader, specified the special comment or tags.
package json

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// user can directly use this to UnMarshal a json stream.
func Unmarshal(r io.Reader, v interface{}) (err error) {
	// read the whole config to []byte.
	var d *json.Decoder

	d = json.NewDecoder(NewJsonPlusReader(r))
	//d = json.NewDecoder(f)

	if err = d.Decode(v); err != nil {
		return
	}
	return
}

// the reader support c++-style comment,
//      block: /* comments */
//      line: // comments
// to filter the comment and got pure raw data.
func NewJsonPlusReader(r io.Reader) io.Reader {
	startMatches := [][]byte{[]byte("'"), []byte("\""), []byte("//"), []byte("/*")}
	endMatches := [][]byte{[]byte("'"), []byte("\""), []byte("\n"), []byte("*/")}
	isComments := []bool{false, false, true, true}
	requiredMatches := []bool{true, true, false, true}
	return NewCommentReader(r, startMatches, endMatches, isComments, requiredMatches)
}

// error when comment not match.
var commentNotMatch = errors.New("comment not match")

// the reader to ignore specified comments or tags.
func NewCommentReader(r io.Reader, startMatches, endMatches [][]byte, isComments, requiredMatches []bool) io.Reader {
	v := &commentReader{
		s: bufio.NewScanner(r),
		b: &bytes.Buffer{},
	}

	v.s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			// read more.
			return 0, nil, nil
		}

		pos, index := firstMatch(data, startMatches)
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

// the reader support comment with start and end chars.
type commentReader struct {
	b *bytes.Buffer
	s *bufio.Scanner
}

// interface io.Reader
func (v *commentReader) Read(p []byte) (n int, err error) {
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

// get the first match in flags.
// @return the matched pos in data and the index of flags.
func firstMatch(data []byte, flags [][]byte) (pos, index int) {
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
