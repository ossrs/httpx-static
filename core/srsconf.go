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

type srsConfState uint8
const (
	ScsInit srsConfState = iota
	ScsText
	ScsNoComment
	ScsComment
)

// the reader support bash-style comment,
//      line: # comments
type srsConfCommentReader struct {
	quotation byte
	st srsConfState
	br *bufio.Reader
}

func NewSrsConfCommentReader(r io.Reader) io.Reader {
	return &srsConfCommentReader{
		br: bufio.NewReader(r),
		st: ScsInit,
	}
}

// interface io.Reader
func (v *srsConfCommentReader) Read(p []byte) (n int, err error) {
	for n < len(p) {
		// from init to working state.
		if v.st == ScsInit {
			var match bool
			if match,err = startsWith(v.br, '#'); err != nil {
				if err == io.EOF {
					v.st = ScsText
					continue
				}
				return
			} else if match {
				v.st = ScsComment
			} else {
				v.st = ScsText
				continue
			}
			if _, err = v.br.Discard(1); err != nil {
				return
			}
		}

		// discard all newline, like \n \r
		if v.st == ScsComment {
			if err = discardUtilAny(v.br, '\n', '\r'); err != nil {
				return
			}
			if err = discardUtilNot(v.br, '\n', '\r'); err != nil {
				return
			}
		}

		// append text.
		if v.st == ScsText || v.st == ScsNoComment {
			var ch byte
			if ch,err = v.br.ReadByte(); err != nil {
				return
			}
			if v.st == ScsText {
				if ch == '"' || ch == '\'' {
					v.quotation = ch
					v.st = ScsNoComment
				}
			} else {
				if ch == v.quotation {
					v.st = ScsText
				}
			}
			p[n] = ch
			n++
		}

		// reset to init state.
		if v.st != ScsNoComment {
			v.st = ScsInit
		}
	}
	return
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

