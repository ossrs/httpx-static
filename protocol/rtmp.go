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
	"fmt"
	"github.com/ossrs/go-oryx/core"
	"io"
	"time"
)

// use []byte as io.Writer
type bytesWriter struct {
	pos int
	b   []byte
}

func NewBytesWriter(b []byte) io.Writer {
	return &bytesWriter{
		b: b,
	}
}

func (v *bytesWriter) Write(p []byte) (n int, err error) {
	if p == nil || len(p) == 0 {
		return
	}

	// check left space.
	left := len(v.b) - v.pos
	if left < len(p) {
		return 0, fmt.Errorf("overflow, left is %v, requires %v", left, len(p))
	}

	// copy content to left space.
	_ = copy(v.b[v.pos:], p)

	// copy ok.
	n = len(p)
	v.pos += n

	return
}

// bytes for handshake.
type hsBytes struct {
	// whether the
	c0c1Ok   bool
	s0s1s2Ok bool
	c2Ok     bool

	// 1 + 1536 + 1536 = 3073
	c0c1c2 []byte
	// 1 + 1536 + 1536 = 3073
	s0s1s2 []byte
}

func NewHsBytes() *hsBytes {
	return &hsBytes{
		c0c1c2: make([]byte, 3073),
		s0s1s2: make([]byte, 3073),
	}
}

func (v *hsBytes) C0() []byte {
	return v.c0c1c2[:1]
}

func (v *hsBytes) C1() []byte {
	return v.c0c1c2[1:1537]
}

func (v *hsBytes) C0C1() []byte {
	return v.c0c1c2[:1537]
}

func (v *hsBytes) C0C1C2() []byte {
	return v.c0c1c2[:]
}

func (v *hsBytes) C2() []byte {
	return v.c0c1c2[1537:]
}

func (v *hsBytes) S0() []byte {
	return v.s0s1s2[:1]
}

func (v *hsBytes) S1() []byte {
	return v.s0s1s2[1:1537]
}

func (v *hsBytes) S0S2() []byte {
	return v.s0s1s2[:1537]
}

func (v *hsBytes) S2() []byte {
	return v.s0s1s2[1537:]
}

func (v *hsBytes) S0S1S2() []byte {
	return v.s0s1s2[:]
}

func (v *hsBytes) ClientPlaintext() bool {
	return v.C0()[0] == 0x03
}

func (v *hsBytes) ServerPlaintext() bool {
	return v.S0()[0] == 0x03
}

func (v *hsBytes) readC0C1(r io.Reader) (err error) {
	if v.c0c1Ok {
		return
	}

	w := NewBytesWriter(v.C0C1())
	if _, err = io.CopyN(w, r, 1537); err != nil {
		core.Error.Println("read c0c1 failed. err is", err)
		return
	}

	v.c0c1Ok = true
	core.Info.Println("read c0c1 ok.")
	return
}

func (v *hsBytes) readC2(r io.Reader) (err error) {
	if v.c2Ok {
		return
	}

	w := NewBytesWriter(v.C2())
	if _, err = io.CopyN(w, r, 1536); err != nil {
		core.Error.Println("read c2 failed. err is", err)
		return
	}

	v.c2Ok = true
	core.Info.Println("read c2 ok.")
	return
}

func (v *hsBytes) s1Time1() []byte {
	return v.S1()[0:4]
}

func (v *hsBytes) s1Time2() []byte {
	return v.S1()[4:8]
}

func (v *hsBytes) c1Time() []byte {
	return v.C1()[0:4]
}

func (v *hsBytes) createS0S1S2() {
	if v.s0s1s2Ok {
		return
	}

	core.RandomFill(v.S0S1S2())

	// s0
	v.S0()[0] = 0x03

	// s1 time1
	binary.BigEndian.PutUint32(v.s1Time1(), uint32(time.Now().Unix()))

	// s1 time2 copy from c1
	if v.c0c1Ok {
		_ = copy(v.s1Time2(), v.c1Time())
	}

	// if c1 specified, copy c1 to s2.
	// @see: https://github.com/ossrs/srs/issues/46
	_ = copy(v.S2(), v.C1())
}

// rtmp request.
type RtmpRequest struct {
	// the tcUrl in RTMP connect app request.
	TcUrl string
}

// rtmp protocol stack.
type Rtmp struct {
	handshake *hsBytes
	transport io.ReadWriter
}

func NewRtmp(transport io.ReadWriter) *Rtmp {
	return &Rtmp{
		handshake: NewHsBytes(),
		transport: transport,
	}
}

func (v *Rtmp) Handshake() (err error) {
	// read c0c2
	if err = v.handshake.readC0C1(v.transport); err != nil {
		return
	}

	// plain text required.
	if !v.handshake.ClientPlaintext() {
		return fmt.Errorf("only support rtmp plain text.")
	}

	// create s0s1s2 from c1.
	v.handshake.createS0S1S2()

	// write s0s1s2 to client.
	r := bytes.NewReader(v.handshake.S0S1S2())
	if _, err = io.CopyN(v.transport, r, 3073); err != nil {
		return
	}

	// read c2
	if err = v.handshake.readC2(v.transport); err != nil {
		return
	}

	return
}

func (v *Rtmp) ConnectApp() (r *RtmpRequest, err error) {
	r = &RtmpRequest{}

	// TODO: FIXME: implements it.
	return
}
