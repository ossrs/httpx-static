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
	"io"
	"bufio"
)

type srsConfDirective struct {
	name string
	args []string
	directives []*srsConfDirective
}

func NewSrsConfDirective() *srsConfDirective {
	return &srsConfDirective{
		args: make([]string, 0),
		directives: make([]*srsConfDirective, 0),
	}
}

func (v *srsConfDirective) ParseDirectives(s *bufio.Scanner) (err error) {
	return
}

// the reader support bash-style comment,
//      line: # comments
func NewSrsConfCommentReader(r io.Reader) io.Reader {
	startMatches := [][]byte{ []byte("'"), []byte("\""), []byte("#"), }
	endMatches := [][]byte{ []byte("'"), []byte("\""), []byte("\n"), }
	isComments := []bool { false, false, true, }
	return NewCommendReader(r, startMatches, endMatches, isComments)
}

// parse the srs style config.
type srsConfParser struct {
	r io.Reader
}

func NewSrsConfParser(r io.Reader) *srsConfParser {
	return &srsConfParser{
		r: NewSrsConfCommentReader(r),
	}
}

func (v *srsConfParser) Decode(c *Config) (err error) {
	scan := bufio.NewScanner(v.r)
	scan.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error){
		return bufio.ScanWords(data, atEOF)
	})

	root := NewSrsConfDirective()
	if err = root.ParseDirectives(scan); err != nil {
		return
	}
	if err = scan.Err(); err != nil {
		return
	}

	// TODO: FIXME: directive root to config c.
	return
}

