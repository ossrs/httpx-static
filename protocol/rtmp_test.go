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
	"testing"
)

func TestBytesWriter(t *testing.T) {
	w := NewBytesWriter(make([]byte, 1))
	if n, err := w.Write(make([]byte, 1)); err != nil || n != 1 {
		t.Error("should be ok")
	}
	if n, err := w.Write(make([]byte, 0)); err != nil || n != 0 {
		t.Error("should be ok")
	}
	if n, err := w.Write(nil); err != nil || n != 0 {
		t.Error("should be ok")
	}
	if n, err := w.Write(make([]byte, 2)); err == nil || n != 0 {
		t.Error("should not be ok")
	}

	w = NewBytesWriter(make([]byte, 10))
	if n, err := w.Write(make([]byte, 5)); err != nil || n != 5 {
		t.Error("should be ok")
	}
	if n, err := w.Write(make([]byte, 5)); err != nil || n != 5 {
		t.Error("should be ok")
	}
	if n, err := w.Write(make([]byte, 1)); err == nil || n != 0 {
		t.Error("should not be ok")
	}
}

func TestHsBytes(t *testing.T) {
	b := NewHsBytes()
	if len(b.c0c1c2) != 3073 {
		t.Error("c0c1c2 should be 3073B")
	}
	if len(b.s0s1s2) != 3073 {
		t.Error("s0s1s2 should be 3073B")
	}
	if len(b.C0()) != 1 {
		t.Error("c0 should be 1B")
	}
	if len(b.C1()) != 1536 {
		t.Error("c1 should be 1536B")
	}
	if len(b.C2()) != 1536 {
		t.Error("c2 should be 1536B")
	}
	if len(b.S0()) != 1 {
		t.Error("s0 should be 1B")
	}
	if len(b.S1()) != 1536 {
		t.Error("s1 should be 1536B")
	}
	if len(b.S2()) != 1536 {
		t.Error("s2 should be 1536B")
	}
}

func TestHsBytes_Plaintext(t *testing.T) {
	b := NewHsBytes()

	b.C0()[0] = 0x03
	if !b.ClientPlaintext() {
		t.Error("should be plaintext")
	}

	b.C0()[0] = 0x04
	if b.ClientPlaintext() {
		t.Error("should not be plaintext")
	}

	b.S0()[0] = 0x03
	if !b.ServerPlaintext() {
		t.Error("should be plaintext")
	}
	b.S0()[0] = 0x04
	if b.ServerPlaintext() {
		t.Error("should not be plaintext")
	}
}

func TestHsBytes_readC0C1(t *testing.T) {
	b := NewHsBytes()

	d := make([]byte, 1537)
	d[0] = 0x0f
	d[1536] = 0x0f
	if err := b.readC0C1(bytes.NewReader(d)); err != nil || !b.c0c1Ok {
		t.Error("should be ok")
	}
	if b.C0()[0] != 0x0f || b.C1()[1535] != 0x0f {
		t.Error("invalid value")
	}

	d = make([]byte, 1536)
	d[0] = 0x0f
	d[1535] = 0x0f
	if err := b.readC2(bytes.NewReader(d)); err != nil || !b.c2Ok {
		t.Error("should be ok")
	}
	if b.C2()[0] != 0x0f || b.C2()[1535] != 0x0f {
		t.Error("invalid value")
	}
}

func TestHsBytes_createS0S1S2(t *testing.T) {
	b := NewHsBytes()

	d := make([]byte, 1537)
	d[0] = 0x0f
	d[1] = 0x0e
	d[2] = 0x0d
	d[3] = 0x0c
	d[4] = 0x0b
	d[1536] = 0x0f
	if err := b.readC0C1(bytes.NewReader(d)); err != nil {
		t.Error("should be ok")
	}

	b.createS0S1S2()
	if b.S0()[0] != 0x03 {
		t.Error("should be plaintext")
	}
	if !bytes.Equal(b.s1Time2(), b.c1Time()) {
		t.Error("invalid time")
	}
	if !bytes.Equal(b.C1(), b.S2()) {
		t.Error("invalid time")
	}
}

func TestRtmpStack_rtmpReadBasicHeader(t *testing.T) {
	fn := func(b []byte, fmt uint8, cid uint32, ef func(error)) {
		r := bytes.NewReader(b)
		if f, c, err := rtmpReadBasicHeader(r); err != nil || f != fmt || c != cid {
			t.Error("invalid chunk,", b, "fmt", fmt, "!=", f, "and cid", cid, "!=", c)
			ef(err)
		}
	}

	fn([]byte{0x02}, 0, 2, func(err error) { t.Error(err) })
	fn([]byte{0x42}, 1, 2, func(err error) { t.Error(err) })
	fn([]byte{0x82}, 2, 2, func(err error) { t.Error(err) })
	fn([]byte{0xC2}, 3, 2, func(err error) { t.Error(err) })

	fn([]byte{0xC2}, 3, 2, func(err error) { t.Error(err) })
	fn([]byte{0xCF}, 3, 0xf, func(err error) { t.Error(err) })
	fn([]byte{0xFF}, 3, 63, func(err error) { t.Error(err) })
	fn([]byte{0xC0, 0x00}, 3, 0+64, func(err error) { t.Error(err) })
	fn([]byte{0xC0, 0x0F}, 3, 0x0f+64, func(err error) { t.Error(err) })
	fn([]byte{0xC0, 0xFF}, 3, 0xff+64, func(err error) { t.Error(err) })
	fn([]byte{0xC1, 0x00, 0x00}, 3, 0*256+0+64, func(err error) { t.Error(err) })
	fn([]byte{0xC1, 0x0F, 0xFF}, 3, 0xff*256+0x0f+64, func(err error) { t.Error(err) })
	fn([]byte{0xC1, 0xFF, 0xFF}, 3, 0xff*256+0xff+64, func(err error) { t.Error(err) })
}

func TestRtmpStack_rtmpReadMessageHeader_extendedTimestamp(t *testing.T) {
	c := NewRtmpChunk(2)
	if b, err := rtmpReadMessageHeader(bytes.NewReader([]byte{
		0xff, 0xff, 0xff,
		0x00, 0x00, 0x0e,
		0x0d,
		0x00, 0x00, 0x00, 0x0c,
		0x00, 0x00, 0x00, 0x0f,
	}), 0, c); err != nil || b != nil {
		t.Error("invalid message")
	}
	if !c.hasExtendedTimestamp || c.timestamp != 0x0f {
		t.Error("invalid timestamp")
	}

	if b, err := rtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x00, 0x0f,
	}), 3, c); err != nil || b != nil {
		t.Error("invalid message")
	}
	if !c.hasExtendedTimestamp || c.timestamp != 0x0f {
		t.Error("invalid timestamp")
	}

	if b, err := rtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x00, 0x0e,
	}), 3, c); err != nil || len(b) != 4 {
		t.Error("invalid message")
	}
	if !c.hasExtendedTimestamp || c.timestamp != 0x0f {
		t.Error("invalid timestamp")
	}

	return
}

func TestRtmpStack_rtmpReadMessageHeader_exceptions(t *testing.T) {
	// fmt is 1, cid 2
	f1 := NewRtmpChunk(2)
	if _, err := rtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0e,
		0x0d,
	}), 1, f1); err != nil {
		t.Error("fresh chunk should ok for fmt=1 and cid=2")
	}

	// fmt is 1, cid not 2
	f1 = NewRtmpChunk(3)
	if _, err := rtmpReadMessageHeader(nil, 1, f1); err == nil {
		t.Error("fresh chunk should error for fmt=1 and cid!=2")
	}

	// fmt is 2
	f2 := NewRtmpChunk(2)
	if _, err := rtmpReadMessageHeader(nil, 3, f2); err == nil {
		t.Error("fresh chunk should error for fmt=2")
	}

	// fmt is 3
	f3 := NewRtmpChunk(2)
	if _, err := rtmpReadMessageHeader(nil, 3, f3); err == nil {
		t.Error("fresh chunk should error for fmt=3")
	}

	// fmt0=>fmt1, change payload length
	c := NewRtmpChunk(2)
	if _, err := rtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0e,
		0x0d,
		0x00, 0x00, 0x00, 0x0c,
	}), 0, c); err != nil {
		t.Error("invalid chunk.")
	}
	if _, err := rtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x0e,
		0x00, 0x00, 0x0e,
		0x0d,
	}), 1, c); err == nil {
		t.Error("fmt1 should never change timestamp delta")
	}
	if _, err := rtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0f,
		0x0d,
	}), 1, c); err == nil {
		t.Error("fmt1 should never change payload length")
	}
	if _, err := rtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0e,
		0x0e,
	}), 1, c); err == nil {
		t.Error("fmt1 should never change message type")
	}
}

func TestRtmpStack_rtmpReadMessageHeader(t *testing.T) {
	fn := func(b []byte, fmt uint8, c *RtmpChunk, f func([]byte, *RtmpChunk), ef func(error)) {
		r := bytes.NewReader(b)
		if b, err := rtmpReadMessageHeader(r, fmt, c); err != nil || b != nil {
			t.Error("error for fmt", fmt)
			ef(err)
		} else {
			if b != nil || c.fmt != fmt {
				t.Error("invalid chunk")
			}

			f(b, c)
		}
	}

	// fmt is 0
	f0 := NewRtmpChunk(2)
	fn([]byte{
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0e,
		0x0d,
		0x00, 0x00, 0x00, 0x0c,
	}, 0, f0, func(b []byte, c *RtmpChunk) {
		if c.payloadLength != 0x0e || c.messageType != 0x0d || c.streamId != 0x0c {
			t.Error("invalid message")
		}
		if c.timestamp != 0x0f || c.timestampDelta != 0x0f {
			t.Error("invalid timestamp")
		}
	}, func(err error) {
		t.Error("invalid message header. err is", err)
	})

	// fmt is 1, cid 2
	f1 := NewRtmpChunk(2)
	fn([]byte{
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0e,
		0x0d,
	}, 1, f1, func(b []byte, c *RtmpChunk) {
		if c.payloadLength != 0x0e || c.messageType != 0x0d {
			t.Error("invalid message")
		}
		if c.timestamp != 0x0f || c.timestampDelta != 0x0f {
			t.Error("invalid timestamp")
		}
	}, func(err error) {
		t.Error("invalid message header. err is", err)
	})

	// fmt0 => fmt1
	fn([]byte{
		0x00, 0x00, 0x03,
		0x00, 0x00, 0x0f,
		0x0e,
	}, 1, f0, func(b []byte, c *RtmpChunk) {
		if c.streamId != 0x0c {
			t.Error("invalid message")
		}
		if c.payloadLength != 0x0f || c.messageType != 0x0e {
			t.Error("invalid message")
		}
		if c.timestamp != 0x0f+0x03 || c.timestampDelta != 0x03 {
			t.Error("invalid timestamp")
		}
	}, func(err error) {
		t.Error("invalid message header. err is", err)
	})

	// fmt0=>fmt1=>fmt2
	fn([]byte{
		0x00, 0x00, 0x05,
	}, 1, f0, func(b []byte, c *RtmpChunk) {
		if c.streamId != 0x0c {
			t.Error("invalid message")
		}
		if c.payloadLength != 0x0f || c.messageType != 0x0e {
			t.Error("invalid message")
		}
		if c.timestamp != 0x0f+0x03+0x05 || c.timestampDelta != 0x05 {
			t.Error("invalid timestamp")
		}
	}, func(err error) {
		t.Error("invalid message header. err is", err)
	})

	// fmt0=>fmt1=>fmt2=>fm3
	fn([]byte{}, 1, f0, func(b []byte, c *RtmpChunk) {
		if c.streamId != 0x0c {
			t.Error("invalid message")
		}
		if c.payloadLength != 0x0f || c.messageType != 0x0e {
			t.Error("invalid message")
		}
		if c.timestamp != 0x0f+0x03+0x05+0x05 || c.timestampDelta != 0x05 {
			t.Error("invalid timestamp")
		}
	}, func(err error) {
		t.Error("invalid message header. err is", err)
	})

	return
}
