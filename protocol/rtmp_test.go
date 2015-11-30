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
	"github.com/ossrs/go-oryx/core"
	"io"
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

func TestRtmpStack(t *testing.T) {
	r := NewRtmpStack(nil, nil)
	if r.inChunkSize != 128 {
		t.Error("default chunk size must be 128, actual is", r.inChunkSize)
	}
}

func TestRtmpStack_RtmpReadBasicHeader(t *testing.T) {
	fn := func(b []byte, fmt uint8, cid uint32, ef func(error)) {
		r := bytes.NewReader(b)
		if f, c, err := RtmpReadBasicHeader(r, &bytes.Buffer{}); err != nil || f != fmt || c != cid {
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

func TestRtmpStack_RtmpReadMessageHeader_extendedTimestamp(t *testing.T) {
	c := NewRtmpChunk(2)
	b := &bytes.Buffer{}
	if err := RtmpReadMessageHeader(bytes.NewReader([]byte{
		0xff, 0xff, 0xff,
		0x00, 0x00, 0x0e,
		0x0d,
		0x0c, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x0f,
	}), b, 0, c); err != nil || b.Len() != 0 {
		t.Error("invalid message")
	}
	if !c.hasExtendedTimestamp || c.timestamp != 0x0f {
		t.Error("invalid timestamp", c.timestamp)
	}

	if err := RtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x00, 0x0f,
	}), b, 3, c); err != nil || b.Len() != 0 {
		t.Error("invalid message")
	}
	if !c.hasExtendedTimestamp || c.timestamp != 0x0f {
		t.Error("invalid timestamp", c.timestamp)
	}

	if err := RtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x00, 0x0e,
	}), b, 3, c); err != nil || b.Len() != 4 {
		t.Error("invalid message")
	}
	if !c.hasExtendedTimestamp || c.timestamp != 0x0f {
		t.Error("invalid timestamp", c.timestamp)
	}

	return
}

func TestRtmpStack_RtmpReadMessageHeader_exceptions(t *testing.T) {
	// fmt is 1, cid 2
	f1 := NewRtmpChunk(2)
	b := &bytes.Buffer{}
	if err := RtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0e,
		0x0d,
	}), b, 1, f1); err != nil {
		t.Error("fresh chunk should ok for fmt=1 and cid=2")
	}

	// fmt is 1, cid not 2
	f1 = NewRtmpChunk(3)
	if err := RtmpReadMessageHeader(nil, b, 1, f1); err == nil {
		t.Error("fresh chunk should error for fmt=1 and cid!=2")
	}

	// fmt is 2
	f2 := NewRtmpChunk(2)
	if err := RtmpReadMessageHeader(nil, b, 3, f2); err == nil {
		t.Error("fresh chunk should error for fmt=2")
	}

	// fmt is 3
	f3 := NewRtmpChunk(2)
	if err := RtmpReadMessageHeader(nil, b, 3, f3); err == nil {
		t.Error("fresh chunk should error for fmt=3")
	}

	// fmt0=>fmt1, change payload length
	c := NewRtmpChunk(2)
	if err := RtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0e,
		0x0d,
		0x0c, 0x00, 0x00, 0x00,
	}), b, 0, c); err != nil {
		t.Error("invalid chunk.")
	}
	if err := RtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x0e,
		0x00, 0x00, 0x0e,
		0x0d,
	}), b, 1, c); err == nil {
		t.Error("fmt1 should never change timestamp delta")
	}
	if err := RtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0f,
		0x0d,
	}), b, 1, c); err == nil {
		t.Error("fmt1 should never change payload length")
	}
	if err := RtmpReadMessageHeader(bytes.NewReader([]byte{
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0e,
		0x0e,
	}), b, 1, c); err == nil {
		t.Error("fmt1 should never change message type")
	}
}

func TestRtmpStack_RtmpReadMessageHeader(t *testing.T) {
	fn := func(b []byte, fmt uint8, c *RtmpChunk, f func([]byte, *RtmpChunk), ef func(error)) {
		r := bytes.NewReader(b)
		if err := RtmpReadMessageHeader(r, &bytes.Buffer{}, fmt, c); err != nil {
			t.Error("error for fmt", fmt, "and cid", c.cid)
			ef(err)
		} else {
			if c.fmt != fmt {
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
		0x0c, 0x00, 0x00, 0x00,
	}, 0, f0, func(b []byte, c *RtmpChunk) {
		if c.partialMessage.Payload.Len() != 0x00 {
			t.Error("invalid payload")
		}
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
		0x00, 0x00, 0x0f,
		0x00, 0x00, 0x0e,
		0x0d,
	}, 1, f0, func(b []byte, c *RtmpChunk) {
		if c.streamId != 0x0c {
			t.Error("invalid message")
		}
		if c.payloadLength != 0x0e || c.messageType != 0x0d {
			t.Error("invalid message")
		}
		if c.timestamp != 0x0f || c.timestampDelta != 0x0f {
			t.Error("invalid timestamp")
		}
	}, func(err error) {
		t.Error("invalid message header. err is", err)
	})

	// fmt0=>fmt1=>fmt2
	fn([]byte{
		0x00, 0x00, 0x0f,
	}, 2, f0, func(b []byte, c *RtmpChunk) {
		if c.streamId != 0x0c {
			t.Error("invalid message")
		}
		if c.payloadLength != 0x0e || c.messageType != 0x0d {
			t.Error("invalid message")
		}
		if c.timestamp != 0x0f || c.timestampDelta != 0x0f {
			t.Error("invalid timestamp")
		}
	}, func(err error) {
		t.Error("invalid message header. err is", err)
	})

	// fmt0=>fmt1=>fmt2=>fm3
	fn([]byte{}, 3, f0, func(b []byte, c *RtmpChunk) {
		if c.streamId != 0x0c {
			t.Error("invalid message")
		}
		if c.payloadLength != 0x0e || c.messageType != 0x0d {
			t.Error("invalid message")
		}
		if c.timestamp != 0x0f || c.timestampDelta != 0x0f {
			t.Error("invalid timestamp")
		}
	}, func(err error) {
		t.Error("invalid message header. err is", err)
	})

	return
}

func TestRtmpChunk_RtmpReadMessagePayload(t *testing.T) {
	c := NewRtmpChunk(2)

	c.partialMessage = NewRtmpMessage()
	if m, err := RtmpReadMessagePayload(0, nil, nil, c); err != nil || m != nil {
		t.Error("should be empty")
	}

	c.partialMessage = NewRtmpMessage()
	c.payloadLength = 2
	if m, err := RtmpReadMessagePayload(2, bytes.NewReader([]byte{
		0x01, 0x02, 0x03, 0x04, 0x05,
	}), &bytes.Buffer{}, c); err != nil || m == nil {
		t.Error("invalid msg")
	}

	c.partialMessage = NewRtmpMessage()
	c.payloadLength = 2
	if m, err := RtmpReadMessagePayload(2, nil, bytes.NewBuffer([]byte{
		0x01, 0x02,
	}), c); err != nil || m == nil {
		t.Error("invalid msg")
	}

	c.partialMessage = NewRtmpMessage()
	c.payloadLength = 5
	if m, err := RtmpReadMessagePayload(5, bytes.NewReader([]byte{
		0x01, 0x02,
	}), bytes.NewBuffer([]byte{
		0x03, 0x04, 0x05,
	}), c); err != nil || m == nil {
		t.Error("invalid msg")
	}

	c.partialMessage = NewRtmpMessage()
	c.payloadLength = 5
	if m, err := RtmpReadMessagePayload(6, bytes.NewReader([]byte{
		0x01, 0x02,
	}), bytes.NewBuffer([]byte{
		0x03, 0x04, 0x05,
	}), c); err != nil || m == nil {
		t.Error("invalid msg")
	}

	c.partialMessage = NewRtmpMessage()
	c.payloadLength = 5
	if m, err := RtmpReadMessagePayload(2, bytes.NewReader([]byte{
		0x01, 0x02, 0x05,
	}), bytes.NewBuffer([]byte{
		0x03, 0x04,
	}), c); err != nil || m != nil {
		t.Error("invalid msg")
	}
	if m, err := RtmpReadMessagePayload(2, bytes.NewReader([]byte{
		0x01, 0x02, 0x05,
	}), &bytes.Buffer{}, c); err != nil || m != nil {
		t.Error("invalid msg")
	}
	if m, err := RtmpReadMessagePayload(2, bytes.NewReader([]byte{
		0x01, 0x02, 0x05,
	}), &bytes.Buffer{}, c); err != nil || m == nil {
		t.Error("invalid msg")
	}
}

func TestMixReader(t *testing.T) {
	r := NewMixReader(nil, nil)
	if _, err := r.Read(make([]byte, 1)); err == nil {
		t.Error("nil source should failed.")
	}

	r = NewMixReader(bytes.NewBuffer([]byte{0x00}), bytes.NewReader([]byte{0x00}))
	if _, err := io.CopyN(NewBytesWriter(make([]byte, 2)), r, 2); err != nil {
		t.Error("should not be nil")
	}
	if _, err := r.Read(make([]byte, 1)); err == nil {
		t.Error("should dry")
	}

	r = NewMixReader(bytes.NewBuffer([]byte{0x00}), bytes.NewReader([]byte{0x00}))
	if _, err := io.CopyN(NewBytesWriter(make([]byte, 1)), r, 1); err != nil {
		t.Error("should not be nil")
	}
	if _, err := io.CopyN(NewBytesWriter(make([]byte, 1)), r, 1); err != nil {
		t.Error("should not be nil")
	}
	if _, err := r.Read(make([]byte, 1)); err == nil {
		t.Error("should dry")
	}
}

func TestRtmpConnectAppPacket(t *testing.T) {
	p := NewRtmpConnectAppPacket().(*RtmpConnectAppPacket)
	if p.Name != Amf0String("connect") || p.TransactionId != Amf0Number(1.0) || p.Args != nil {
		t.Error("invalid connect app packet.")
	}

	p = NewRtmpConnectAppPacket().(*RtmpConnectAppPacket)
	if err := p.UnmarshalBinary([]byte{
		2, 0, 7, 'c', 'o', 'n', 'n', 'e', 'c', 't', // string: connect
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0, // number: 1.0
		3,
		0, 2, 'p', 'j', 2, 0, 4, 'o', 'r', 'y', 'x', // "pj"=string("oryx")
		0, 6, 'c', 'r', 'e', 'a', 't', 'e', 0, 0x40, 0x9f, 0x7c, 0, 0, 0, 0, 0, // "create"=number(2015)
		0, 0, 9, // object
	}); err != nil || p.Name != Amf0String("connect") || p.TransactionId != Amf0Number(1.0) {
		t.Error("invalid packet, err is", err)
	}
	if v, ok := p.CommandObject.Get("pj").(*Amf0String); !ok || *v != Amf0String("oryx") {
		t.Error("invalid packet. v is", v)
	}
	if p.Args != nil {
		t.Error("invalid packet.")
	}

	p = NewRtmpConnectAppPacket().(*RtmpConnectAppPacket)
	if err := p.UnmarshalBinary([]byte{
		2, 0, 7, 'c', 'o', 'n', 'n', 'e', 'c', 't', // string: connect
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0, // number: 1.0
		3,
		0, 2, 'p', 'j', 2, 0, 4, 'o', 'r', 'y', 'x', // "pj"=string("oryx")
		0, 6, 'c', 'r', 'e', 'a', 't', 'e', 0, 0x40, 0x9f, 0x7c, 0, 0, 0, 0, 0, // "create"=number(2015)
		0, 0, 9, // object
		3,
		0, 7, 'v', 'e', 'r', 's', 'i', 'o', 'n', 2, 0, 3, '1', '.', '0', // "version"=string("1.0")
		0, 0, 9, // object
	}); err != nil {
		t.Error("invalid packet, err is", err)
	}
	if p.Args == nil {
		t.Error("invalid packet.")
	} else if v, ok := p.Args.Get("version").(*Amf0String); !ok || *v != Amf0String("1.0") {
		t.Error("invalid packet. v is", v)
	}

	p = NewRtmpConnectAppPacket().(*RtmpConnectAppPacket)
	p.CommandObject.Set("pj", NewAmf0String("oryx"))
	p.CommandObject.Set("create", NewAmf0Number(2015))
	size := 3 + 7 + 1 + 8 + 1 + 3 + 2 + 2 + 3 + 4 + 2 + 6 + 1 + 8
	if b, err := p.MarshalBinary(); err != nil || len(b) != size {
		t.Error("invalid packet, size is", size, "b is", len(b))
	}

	p.Args = NewAmf0Object()
	p.Args.Set("version", NewAmf0String("1.0"))
	size += 1 + 3 + 2 + 7 + 3 + 3
	if b, err := p.MarshalBinary(); err != nil || len(b) != size {
		t.Error("invalid packet, size is", size, "b is", len(b))
	}
}

func TestRtmpSetWindowAckSizePacket(t *testing.T) {
	p := NewRtmpSetWindowAckSizePacket().(*RtmpSetWindowAckSizePacket)
	if p.Ack != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{0, 0, 0, 0x0f}); err != nil || p.Ack != 0xf {
		t.Error("invalid, err is", err)
	}

	p = NewRtmpSetWindowAckSizePacket().(*RtmpSetWindowAckSizePacket)
	p.Ack = 0xff
	if b, err := p.MarshalBinary(); err != nil || len(b) != 4 {
		t.Error("invalid, err is", err)
	}
}

func TestRtmpUserControlPacket(t *testing.T) {
	p := NewRtmpUserControlPacket().(*RtmpUserControlPacket)
	if p.EventType != 0 || p.EventData != 0 || p.ExtraData != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{0, 0, 0, 0, 0, 1}); err != nil || p.EventType != 0 || p.EventData != 1 {
		t.Error("invalid, err is", err)
	}

	p = NewRtmpUserControlPacket().(*RtmpUserControlPacket)
	p.EventType = RtmpUint16(RtmpPcucSetBufferLength)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 10 {
		t.Error("invalid, err is", err)
	}

	p = NewRtmpUserControlPacket().(*RtmpUserControlPacket)
	p.EventType = RtmpUint16(RtmpPcucStreamIsRecorded)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 6 {
		t.Error("invalid, err is", err)
	}
}

func TestRtmpSetChunkSizePacket(t *testing.T) {
	p := NewRtmpSetChunkSizePacket().(*RtmpSetChunkSizePacket)
	if p.ChunkSize != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{0, 0, 0, 0x0f}); err != nil || p.ChunkSize != 0xf {
		t.Error("invalid, err is", err)
	}

	p = NewRtmpSetChunkSizePacket().(*RtmpSetChunkSizePacket)
	p.ChunkSize = 0xff
	if b, err := p.MarshalBinary(); err != nil || len(b) != 4 {
		t.Error("invalid, err is", err)
	}
}

func TestRtmpSetPeerBandwidthPacket(t *testing.T) {
	p := NewRtmpSetPeerBandwidthPacket().(*RtmpSetPeerBandwidthPacket)
	if p.Bandwidth != 0 || p.Type != RtmpUint8(2) {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{0, 0, 0, 0x0f, 1}); err != nil || p.Bandwidth != 0xf || p.Type != RtmpUint8(1) {
		t.Error("invalid, err is", err)
	}

	p = NewRtmpSetPeerBandwidthPacket().(*RtmpSetPeerBandwidthPacket)
	p.Bandwidth = 0xff
	p.Type = RtmpUint8(Soft)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 5 {
		t.Error("invalid, err is", err)
	}
}

func TestRtmpConnectAppResPacket(t *testing.T) {
	p := NewRtmpConnectAppResPacket().(*RtmpConnectAppResPacket)
	if p.TransactionId != 1.0 || p.Name != "_result" {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		3,
		0, 2, 'p', 'j', 2, 0, 4, 'o', 'r', 'y', 'x',
		0, 0, 9, // object
		3,
		0, 6, 'c', 'r', 'e', 'a', 't', 'e', 0, 0x40, 0x9f, 0x7c, 0, 0, 0, 0, 0,
		0, 0, 9, //object
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" || p.TransactionId != 1.0 {
		t.Error("invalid")
	}
	if v, ok := p.Props.Get("pj").(*Amf0String); !ok || *v != "oryx" {
		t.Error("invalid")
	}
	if v, ok := p.Info.Get("create").(*Amf0Number); !ok || *v != 2015 {
		t.Error("invalid")
	}

	p = NewRtmpConnectAppResPacket().(*RtmpConnectAppResPacket)
	p.Props.Set("pj", NewAmf0String("oryx"))
	p.Info.Set("create", NewAmf0Number(2015))
	if b, err := p.MarshalBinary(); err != nil || len(b) != 55 {
		t.Error("invalid")
	}
}

func TestRtmpOnBwDonePacket(t *testing.T) {
	p := NewRtmpOnBwDonePacket().(*RtmpOnBwDonePacket)
	if p.Name != "onBWDone" || p.TransactionId != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		5,
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" || p.TransactionId != 1.0 {
		t.Error("invalid")
	}

	p = NewRtmpOnBwDonePacket().(*RtmpOnBwDonePacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 21 {
		t.Error("invalid")
	}
}

func TestRtmpCreateStreamPacket(t *testing.T) {
	p := NewRtmpCreateStreamPacket().(*RtmpCreateStreamPacket)
	if p.Name != "createStream" || p.TransactionId != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		5,
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" || p.TransactionId != 1.0 {
		t.Error("invalid")
	}

	p = NewRtmpCreateStreamPacket().(*RtmpCreateStreamPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 25 {
		t.Error("invalid")
	}
}

func TestRtmpCreateStreamResPacket(t *testing.T) {
	p := NewRtmpCreateStreamResPacket().(*RtmpCreateStreamResPacket)
	if p.Name != "_result" || p.TransactionId != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		5,
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" || p.TransactionId != 1.0 || p.StreamId != 1.0 {
		t.Error("invalid")
	}

	p = NewRtmpCreateStreamResPacket().(*RtmpCreateStreamResPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 29 {
		t.Error("invalid")
	}
}

func TestRtmpEmptyPacket(t *testing.T) {
	p := NewRtmpEmptyPacket().(*RtmpEmptyPacket)
	if err := p.UnmarshalBinary([]byte{0, 0x40, 0x59, 0, 0, 0, 0, 0, 0}); err != nil || p.Id != 100 {
		t.Error("invalid")
	}
	if b, err := p.MarshalBinary(); err != nil || len(b) != 9 {
		t.Error("invalid")
	}
}

func TestRtmpFMLEStartPacket(t *testing.T) {
	p := NewRtmpFMLEStartPacket().(*RtmpFMLEStartPacket)
	if p.Name != "releaseStream" || p.TransactionId != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		5,
		2, 0, 4, 'o', 'r', 'y', 'x',
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" || p.TransactionId != 1.0 || p.Stream != "oryx" {
		t.Error("invalid")
	}

	p = NewRtmpFMLEStartPacket().(*RtmpFMLEStartPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 29 {
		t.Error("invalid")
	}
}

func TestRtmpFMLEStartResPacket(t *testing.T) {
	p := NewRtmpFMLEStartResPacket().(*RtmpFMLEStartResPacket)
	if p.Name != "_result" || p.TransactionId != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		5,
		6,
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" || p.TransactionId != 1.0 {
		t.Error("invalid")
	}

	p = NewRtmpFMLEStartResPacket().(*RtmpFMLEStartResPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 21 {
		t.Error("invalid")
	}
}

func TestRtmpPlayPacket(t *testing.T) {
	p := NewRtmpPlayPacket().(*RtmpPlayPacket)
	if p.Name != "play" || p.TransactionId != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		5,
		2, 0, 4, 'o', 'r', 'y', 'x',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		1, 1,
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" || p.TransactionId != 1.0 || p.Stream != "oryx" || *p.Start != 1.0 || *p.Duration != 1.0 || *p.Reset != true {
		t.Error("invalid")
	}

	p = NewRtmpPlayPacket().(*RtmpPlayPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 20 {
		t.Error("invalid")
	}
}

func TestRtmpCallPacket(t *testing.T) {
	p := NewRtmpCallPacket().(*RtmpCallPacket)
	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		2, 0, 4, 'o', 'r', 'y', 'x',
		2, 0, 4, 'o', 'r', 'y', 'x',
	}); err != nil {
		t.Error("invalid")
	}
	if p.Name != "_name" || p.TransactionId != 1.0 {
		t.Error("invalid")
	}
	if p, ok := p.Command.(*Amf0String); !ok || *p != "oryx" {
		t.Error("invalid")
	}
	if p, ok := p.Args.(*Amf0String); !ok || *p != "oryx" {
		t.Error("invalid")
	}

	p = NewRtmpCallPacket().(*RtmpCallPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 12 {
		t.Error("invalid")
	}
}

func TestRtmpCallResPacket(t *testing.T) {
	p := NewRtmpCallResPacket().(*RtmpCallResPacket)
	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		2, 0, 4, 'o', 'r', 'y', 'x',
		2, 0, 4, 'o', 'r', 'y', 'x',
	}); err != nil {
		t.Error("invalid")
	}
	if p.Name != "_name" || p.TransactionId != 1.0 {
		t.Error("invalid")
	}
	if p, ok := p.Command.(*Amf0String); !ok || *p != "oryx" {
		t.Error("invalid")
	}
	if p, ok := p.Args.(*Amf0String); !ok || *p != "oryx" {
		t.Error("invalid")
	}

	p = NewRtmpCallResPacket().(*RtmpCallResPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 19 {
		t.Error("invalid")
	}
}

func TestRtmpPublishPacket(t *testing.T) {
	p := NewRtmpPublishPacket().(*RtmpPublishPacket)
	if p.Name != "publish" || p.TransactionId != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		5,
		2, 0, 4, 'o', 'r', 'y', 'x',
		2, 0, 4, 'l', 'i', 'v', 'e',
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" || p.TransactionId != 1.0 || p.Stream != "oryx" || *p.Type != "live" {
		t.Error("invalid")
	}

	p = NewRtmpPublishPacket().(*RtmpPublishPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 23 {
		t.Error("invalid")
	}
}

func TestRtmpOnStatusCallPacket(t *testing.T) {
	p := NewRtmpOnStatusCallPacket().(*RtmpOnStatusCallPacket)
	if p.Name != "onStatus" || p.TransactionId != 0 {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		0, 0x3f, 0xf0, 0, 0, 0, 0, 0, 0,
		5,
		3, 0, 0, 9,
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" || p.TransactionId != 1.0 {
		t.Error("invalid")
	}

	p = NewRtmpOnStatusCallPacket().(*RtmpOnStatusCallPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 25 {
		t.Error("invalid")
	}
}

func TestRtmpSampleAccessPacket(t *testing.T) {
	p := NewRtmpSampleAccessPacket().(*RtmpSampleAccessPacket)
	if p.Name != "|RtmpSampleAccess" || p.VideoSampleAccess != false || p.AudioSampleAccess != false {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		1, 1,
		1, 1,
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" || p.VideoSampleAccess != true || p.AudioSampleAccess != true {
		t.Error("invalid")
	}

	p = NewRtmpSampleAccessPacket().(*RtmpSampleAccessPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 24 {
		t.Error("invalid")
	}
}

func TestRtmpOnStatusDataPacket(t *testing.T) {
	p := NewRtmpOnStatusDataPacket().(*RtmpOnStatusDataPacket)
	if p.Name != "onStatus" {
		t.Error("invalid")
	}

	if err := p.UnmarshalBinary([]byte{
		2, 0, 5, '_', 'n', 'a', 'm', 'e',
		3, 0, 0, 9,
	}); err != nil {
		t.Error("invalid")
	}

	if p.Name != "_name" {
		t.Error("invalid")
	}

	p = NewRtmpOnStatusDataPacket().(*RtmpOnStatusDataPacket)
	if b, err := p.MarshalBinary(); err != nil || len(b) != 15 {
		t.Error("invalid", b)
	}
}

func TestRtmpRequest(t *testing.T) {
	p := NewRtmpRequest()
	if p.TcUrl != "" || p.Stream != "" || p.App != "" || p.Type != RtmpUnknown {
		t.Error("invalid")
	}

	if err := p.Reparse(); err == nil {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip"
	p.Stream = "xxx"
	if err := p.Reparse(); err != nil || p.App != "__defaultApp__" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app___vhost=xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.App != "app" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app?vhost___xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.App != "app" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app?vhost=xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.App != "app" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app...vhost=xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.App != "app" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app?vhost...xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.App != "app" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app"
	p.Stream = "stream?vhost=xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.Stream != "stream" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app"
	p.Stream = "stream?domain=xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.Stream != "stream" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app"
	p.Stream = "stream...domain=xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.Stream != "stream" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app"
	p.Stream = "stream?domain...xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.Stream != "stream" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app"
	p.Stream = "stream___domain=xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.Stream != "stream" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip/app"
	p.Stream = "stream?domain___xx"
	if err := p.Reparse(); err != nil || p.Vhost != "xx" || p.Stream != "stream" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://vhost/app"
	p.Stream = "stream"
	if err := p.Reparse(); err != nil || p.Vhost != "vhost" || p.App != "app" || p.Stream != "stream" || p.Port() != 1935 {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip:1936/app"
	p.Stream = "stream"
	if err := p.Reparse(); err != nil || p.Vhost != "ip" || p.App != "app" || p.Stream != "stream" || p.Port() != 1936 || p.Host() != "ip" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip:1936/app?vhost=xxx"
	p.Stream = "stream?vhost=xxx"
	if err := p.Reparse(); err != nil || p.Vhost != "xxx" {
		t.Error("invalid")
	}

	p.TcUrl = "rtmp://ip:1936/app/sub"
	if err := p.Reparse(); err != nil || p.App != "app/sub" {
		t.Error("invalid")
	}
}

func TestRtmpMessage(t *testing.T) {
	var err error
	m := NewRtmpMessage()

	if _, err = m.Payload.Write([]byte{7, 9, 3, 4, 2}); err != nil {
		t.Error("invalid", err)
	}

	b := m.Payload.Bytes()
	if b[0] != 7 || b[4] != 2 {
		t.Error("invalid")
	}

	if m.Timestamp = 100; m.Timestamp != 100 {
		t.Error("invalid")
	}

	var o core.Message
	if o, err = m.ToMessage(); err != nil {
		t.Error("invalid", err)
	}

	var ok bool
	var om *OryxRtmpMessage
	if om, ok = o.(*OryxRtmpMessage); !ok {
		t.Error("invalid")
	}
	if om.Timestamp() != 100 {
		t.Error("invalid")
	}

	cp := om.Copy()
	if cp.Timestamp() != 100 {
		t.Error("invalid")
	}

	if cp.SetTimestamp(200); cp.Timestamp() != 200 || om.Timestamp() != 100 {
		t.Error("invalid")
	}

	if cp.rtmp.Payload.Bytes()[0] != 7 {
		t.Error("invalid")
	}
	if om.rtmp.Payload.Bytes()[0] != 7 {
		t.Error("invalid")
	}

	// the Buffer shares the bytes.
	if cp.rtmp.Payload.Bytes()[0] = 8; cp.rtmp.Payload.Bytes()[0] != 8 {
		t.Error("invalid")
	}
	if om.rtmp.Payload.Bytes()[0] != 8 {
		t.Error("invalid")
	}

	// when write, the Buffer copy it.
	if _, err = cp.rtmp.Payload.Write([]byte{3}); err != nil {
		t.Error("invalid", err)
	}
	if cp.rtmp.Payload.Len() != 6 || cp.rtmp.Payload.Bytes()[5] != 3 {
		t.Error("invalid")
	}
	if om.rtmp.Payload.Len() != 5 {
		t.Error("invalid")
	}
}

func TestComplexHandshake(t *testing.T) {
	k := chsKey(make([]byte, 764))
	k[760] = 1
	k[761] = 2
	k[762] = 3
	k[763] = 4
	if k.Offset() != 10 {
		t.Error("invalid")
	}
	if len(k.Random0()) != 10 || len(k.Key()) != 128 || len(k.Random1()) != 622 {
		t.Error("invalid")
	}

	k[760] = 0
	k[761] = 0
	k[762] = 0
	k[763] = 0
	if k.Offset() != 0 {
		t.Error("invalid")
	}
	if len(k.Random0()) != 0 || len(k.Key()) != 128 || len(k.Random1()) != 632 {
		t.Error("invalid")
	}

	k[760] = 31
	k[761] = 200
	k[762] = 200
	k[763] = 200
	if k.Offset() != 631 {
		t.Error("invalid")
	}
	if len(k.Random0()) != 631 || len(k.Key()) != 128 || len(k.Random1()) != 1 {
		t.Error("invalid")
	}

	k[760] = 32
	k[761] = 200
	k[762] = 200
	k[763] = 200
	if k.Offset() != 0 {
		t.Error("invalid")
	}
	if len(k.Random0()) != 0 || len(k.Key()) != 128 || len(k.Random1()) != 632 {
		t.Error("invalid")
	}

	d := chsDigest(make([]byte, 764))
	d[0] = 1
	d[1] = 2
	d[2] = 3
	d[3] = 4
	if d.Offset() != 10 {
		t.Error("invalid")
	}
	if len(d.Random0()) != 10 || len(d.Digest()) != 32 || len(d.Random1()) != 718 {
		t.Error("invalid")
	}

	d[0] = 0
	d[1] = 0
	d[2] = 0
	d[3] = 0
	if d.Offset() != 0 {
		t.Error("invalid")
	}
	if len(d.Random0()) != 0 || len(d.Digest()) != 32 || len(d.Random1()) != 728 {
		t.Error("invalid")
	}

	d[0] = 27
	d[1] = 250
	d[2] = 250
	d[3] = 200
	if d.Offset() != 727 {
		t.Error("invalid")
	}
	if len(d.Random0()) != 727 || len(d.Digest()) != 32 || len(d.Random1()) != 1 {
		t.Error("invalid")
	}

	d[0] = 28
	d[1] = 250
	d[2] = 250
	d[3] = 200
	if d.Offset() != 0 {
		t.Error("invalid")
	}
	if len(d.Random0()) != 0 || len(d.Digest()) != 32 || len(d.Random1()) != 728 {
		t.Error("invalid")
	}

	c1 := &chsC1S1{}
	if err := c1.Parse([]byte{}, Schema0); err == nil {
		t.Error("invalid")
	}
}

func TestSha256(t *testing.T) {
	// randome bytes to ensure the openssl sha256 is ok.
	b := []byte{
		0x8b, 0x1c, 0x5c, 0x5c, 0x3b, 0x98, 0x60, 0x80, 0x3c, 0x97, 0x43, 0x79, 0x9c, 0x94, 0xec, 0x63, 0xaa, 0xd9, 0x10, 0xd7, 0x0d, 0x91, 0xfb, 0x1f, 0xbf, 0xe0, 0x29, 0xde, 0x77, 0x09, 0x21, 0x34, 0xa5, 0x7d, 0xdf, 0xe3, 0xdf, 0x11, 0xdf, 0xd4, 0x00, 0x57, 0x38, 0x5b, 0xae, 0x9e, 0x89, 0x35, 0xcf, 0x07, 0x48, 0xca, 0xc8, 0x25, 0x46, 0x3c,
		0xb6, 0xdb, 0x9b, 0x39, 0xa6, 0x07, 0x3d, 0xaf, 0x8b, 0x85, 0xa2, 0x2f, 0x03, 0x64, 0x5e, 0xbd, 0xb4, 0x20, 0x01, 0x48, 0x2e, 0xc2, 0xe6, 0xcc, 0xce, 0x61, 0x59, 0x47, 0xf9, 0xdd, 0xc2, 0xa2, 0xfe, 0x64, 0xe6, 0x0b, 0x41, 0x4f, 0xe4, 0x8a, 0xca, 0xbe, 0x4d, 0x0e, 0x73, 0xba, 0x82, 0x30, 0x3c, 0x53, 0x36, 0x2e, 0xd3, 0x04, 0xae, 0x49,
		0x44, 0x71, 0x6d, 0x4d, 0x5a, 0x14, 0x94, 0x94, 0x57, 0x78, 0xb9, 0x2a, 0x34, 0x49, 0xf8, 0xc2, 0xec, 0x4e, 0x29, 0xb6, 0x28, 0x54, 0x4a, 0x5e, 0x68, 0x06, 0xfe, 0xfc, 0xd5, 0x01, 0x35, 0x0c, 0x95, 0x6f, 0xe9, 0x77, 0x8a, 0xfc, 0x11, 0x15, 0x1a, 0xda, 0x6c, 0xf5, 0xba, 0x9e, 0x41, 0xd9, 0x7e, 0x0f, 0xdb, 0x33, 0xda, 0x35, 0x9d, 0x34,
		0x67, 0x8f, 0xdf, 0x71, 0x63, 0x04, 0x9c, 0x54, 0xb6, 0x18, 0x10, 0x2d, 0x42, 0xd2, 0xf3, 0x14, 0x34, 0xa1, 0x31, 0x90, 0x48, 0xc9, 0x4b, 0x87, 0xb5, 0xcd, 0x62, 0x6b, 0x77, 0x18, 0x36, 0xd9, 0xc9, 0xc9, 0xae, 0x89, 0xfb, 0xed, 0xcd, 0xcb, 0xdb, 0x6e, 0xe3, 0x22, 0xbf, 0x7b, 0x72, 0x8a, 0xc3, 0x79, 0xd6, 0x1b, 0x6c, 0xe7, 0x9c, 0xc9,
		0xfd, 0x48, 0xaa, 0xc1, 0xfa, 0xbf, 0x33, 0x87, 0x5c, 0x0d, 0xe5, 0x34, 0x24, 0x70, 0x14, 0x1e, 0x4a, 0x48, 0x07, 0x6e, 0xaf, 0xbf, 0xfe, 0x34, 0x1e, 0x1e, 0x19, 0xfc, 0xb5, 0x8a, 0x4f, 0x3c, 0xb4, 0xcf, 0xde, 0x24, 0x79, 0x65, 0x17, 0x22, 0x3f, 0xc0, 0x06, 0x76, 0x4e, 0x3c, 0xfb, 0xc3, 0xd0, 0x7f, 0x7b, 0x87, 0x5c, 0xeb, 0x97, 0x87,
	}

	es := []byte{
		0x1b, 0xc7, 0xe6, 0x14, 0xd5, 0x19, 0x8d, 0x99, 0x42, 0x0a, 0x21, 0x95, 0x26, 0x9a, 0x8a, 0x56,
		0xb4, 0x82, 0x2a, 0x7f, 0xd3, 0x1d, 0xc3, 0xd8, 0x92, 0x97, 0xc4, 0x61, 0xb7, 0x4d, 0x5d, 0xd2,
	}

	if s, err := opensslHmacSha256(RtmpGenuineFPKey[0:30], b); err != nil {
		t.Error("invalid")
	} else if !bytes.Equal(s, es) {
		t.Error("invalid, s is", len(s), "and es is", len(es))
	}
}

func TestDigest(t *testing.T) {
	// c1s1 schema0
	//     key: 764bytes
	//     digest: 764bytes
	b := make([]byte, 1536)
	// 764bytes digest structure
	//     offset: 4bytes
	//     random-data: (offset)bytes
	//     digest-data: 32bytes
	//     random-data: (764-4-offset-32)bytes
	// @see also: http://blog.csdn.net/win_lin/article/details/13006803
	d := b[8+764:]
	d[0] = 0
	d[1] = 0
	d[2] = 0
	d[3] = 1
	d[4] = 7    // r0
	d[5] = 1    // digest start
	d[5+31] = 2 // digest end.
	d[5+32] = 3 //r1

	c1 := &chsC1S1{}
	if err := c1.Parse(b, Schema0); err != nil {
		t.Error("invalid", err)
	}

	if c1.digestOffset() != 8+764+4+1 {
		t.Error("invalid")
	}
	p1 := c1.part1()
	if p1[len(p1)-1] != 7 || len(p1) != 8+764+5 {
		t.Error("invalid")
	}
	p2 := c1.part2()
	if p2[0] != 3 || len(p2) != 764-5-32 {
		t.Error("invalid")
	}

	join := append([]byte{}, p1...)
	join = append(join, p2...)
	if sum, err := opensslHmacSha256(RtmpGenuineFPKey[0:30], join); err != nil {
		t.Error("invalid", sum)
	}

	if c1.digestOffset() != 8+764+4+1 {
		t.Error("invalid")
	}

	s := b[c1.digestOffset() : c1.digestOffset()+32]
	if s[0] != 1 {
		t.Error("invalid", s[0:32])
	}
	if s[31] != 2 {
		t.Error("invalid", s[0:32])
	}
}

func TestFlashPlayerHandshake(t *testing.T) {
	c0c1 := []byte{
		0x03, 0x00, 0x0f, 0x64, 0xd0, 0x80, 0x00, 0x07, 0x02, 0xe6, 0x42, 0xe5, 0x2b, 0xf1, 0x1d, 0x0f, 0x6c, 0xc8, 0x50, 0xf2, 0x06, 0xae, 0xd5, 0x4f, 0xdb, 0xfe, 0x79, 0xc2, 0xef, 0xf5, 0x01, 0x74, 0x4b, 0x5b, 0xe7, 0x37, 0xa3, 0xe0, 0xca, 0xe1, 0x97, 0x07, 0xdb, 0x54, 0x1d, 0x4c, 0x4b, 0xa3, 0xc3, 0x3e, 0xa9, 0xeb, 0xa9, 0x5b, 0x2f, 0x38, 0xa0, 0xa9, 0x98, 0x38, 0x80, 0x1b, 0xfb, 0xa7, 0x04, 0xff, 0xfd, 0x45, 0xfe, 0xfa, 0xc1, 0xe4, 0x1c, 0x77, 0x9a, 0x19, 0x39, 0x34, 0x10, 0x79, 0x12, 0xcf, 0x4e, 0xea, 0x34, 0x7d, 0x88, 0x47, 0xca, 0xf2, 0xb3, 0x09, 0x50, 0xbb, 0xe1, 0x20, 0x9b, 0x25, 0xb0, 0x3c, 0xbc, 0x46, 0x7a, 0x36, 0xb8, 0xc2, 0x4d, 0xd0, 0xf1, 0x20, 0x2a, 0xcc, 0x7a, 0x91, 0xab, 0x0b, 0xb6, 0xc7, 0x09, 0x0d, 0xf1, 0x34, 0x0c, 0x37, 0xbe, 0xad, 0x0e, 0xe3, 0x6b, 0x68, 0x0a, 0x7e, 0xd2, 0xd4, 0xc5, 0x3d, 0xdc, 0xac, 0x28, 0x8b, 0x88, 0xb5, 0x1e, 0xd8, 0x2b, 0x68, 0x72, 0x55, 0x64, 0xa2, 0xa5, 0x69, 0x0a, 0xdb, 0x26, 0xff, 0x63, 0x2d, 0xb8, 0xff, 0xb6, 0x33, 0xd3, 0x9d, 0x5c, 0x46, 0xd6, 0xbf, 0x8b, 0x1c, 0x5c, 0x5c, 0x3b, 0x98, 0x60, 0x80, 0x3c, 0x97, 0x43, 0x79, 0x9c, 0x94, 0xec, 0x63, 0xaa, 0xd9, 0x10, 0xd7, 0x0d, 0x91, 0xfb, 0x1f, 0xbf, 0xe0, 0x29, 0xde, 0x77, 0x09, 0x21, 0x34, 0xa5, 0x7d, 0xdf, 0xe3, 0xdf, 0x11, 0xdf, 0xd4, 0x00, 0x57, 0x38, 0x5b, 0xae, 0x9e, 0x89, 0x35, 0xcf, 0x07, 0x48, 0xca, 0xc8, 0x25, 0x46, 0x3c,
		0xb6, 0xdb, 0x9b, 0x39, 0xa6, 0x07, 0x3d, 0xaf, 0x8b, 0x85, 0xa2, 0x2f, 0x03, 0x64, 0x5e, 0xbd, 0xb4, 0x20, 0x01, 0x48, 0x2e, 0xc2, 0xe6, 0xcc, 0xce, 0x61, 0x59, 0x47, 0xf9, 0xdd, 0xc2, 0xa2, 0xfe, 0x64, 0xe6, 0x0b, 0x41, 0x4f, 0xe4, 0x8a, 0xca, 0xbe, 0x4d, 0x0e, 0x73, 0xba, 0x82, 0x30, 0x3c, 0x53, 0x36, 0x2e, 0xd3, 0x04, 0xae, 0x49, 0x44, 0x71, 0x6d, 0x4d, 0x5a, 0x14, 0x94, 0x94, 0x57, 0x78, 0xb9, 0x2a, 0x34, 0x49, 0xf8, 0xc2, 0xec, 0x4e, 0x29, 0xb6, 0x28, 0x54, 0x4a, 0x5e, 0x68, 0x06, 0xfe, 0xfc, 0xd5, 0x01, 0x35, 0x0c, 0x95, 0x6f, 0xe9, 0x77, 0x8a, 0xfc, 0x11, 0x15, 0x1a, 0xda, 0x6c, 0xf5, 0xba, 0x9e, 0x41, 0xd9, 0x7e, 0x0f, 0xdb, 0x33, 0xda, 0x35, 0x9d, 0x34, 0x67, 0x8f, 0xdf, 0x71, 0x63, 0x04, 0x9c, 0x54, 0xb6, 0x18, 0x10, 0x2d, 0x42, 0xd2, 0xf3, 0x14, 0x34, 0xa1, 0x31, 0x90, 0x48, 0xc9, 0x4b, 0x87, 0xb5, 0xcd, 0x62, 0x6b, 0x77, 0x18, 0x36, 0xd9, 0xc9, 0xc9, 0xae, 0x89, 0xfb, 0xed, 0xcd, 0xcb, 0xdb, 0x6e, 0xe3, 0x22, 0xbf, 0x7b, 0x72, 0x8a, 0xc3, 0x79, 0xd6, 0x1b, 0x6c, 0xe7, 0x9c, 0xc9, 0xfd, 0x48, 0xaa, 0xc1, 0xfa, 0xbf, 0x33, 0x87, 0x5c, 0x0d, 0xe5, 0x34, 0x24, 0x70, 0x14, 0x1e, 0x4a, 0x48, 0x07, 0x6e, 0xaf, 0xbf, 0xfe, 0x34, 0x1e, 0x1e, 0x19, 0xfc, 0xb5, 0x8a, 0x4f, 0x3c, 0xb4, 0xcf, 0xde, 0x24, 0x79, 0x65, 0x17, 0x22, 0x3f, 0xc0, 0x06, 0x76, 0x4e, 0x3c, 0xfb, 0xc3, 0xd0, 0x7f, 0x7b, 0x87, 0x5c, 0xeb, 0x97, 0x87,
		0x99, 0x20, 0x70, 0x7b, 0xf8, 0x97, 0x73, 0xdc, 0xb4, 0x94, 0x43, 0x27, 0x03, 0xbd, 0xb5, 0x91, 0xd9, 0x3e, 0x51, 0x1a, 0xd5, 0x60, 0x9c, 0x71, 0xd3, 0xc7, 0x1f, 0xd7, 0xef, 0x2f, 0xa1, 0xf7, 0xe6, 0xb1, 0x31, 0x9d, 0xec, 0xa3, 0xe1, 0x01, 0x57, 0xa8, 0x1c, 0x34, 0xf8, 0x82, 0xf5, 0x4d, 0xb8, 0x32, 0xe4, 0x4b, 0x90, 0x97, 0xcf, 0x8c, 0x2e, 0x89, 0xd0, 0xbc, 0xc0, 0xca, 0x45, 0x5e, 0x5c, 0x36, 0x47, 0x98, 0xa8, 0x57, 0xb5, 0x56, 0xc9, 0x11, 0xe4, 0x2f, 0xf0, 0x2b, 0x2c, 0xc1, 0x49, 0x1a, 0xfb, 0xdd, 0x89, 0x3f, 0x18, 0x98, 0x78, 0x13, 0x83, 0xf4, 0x30, 0xe2, 0x4e, 0x0e, 0xf4, 0x6c, 0xcb, 0xc6, 0xc7, 0x31, 0xe9, 0x78, 0x74, 0xfd, 0x53, 0x05, 0x4e, 0x7b, 0xd3, 0x9b, 0xeb, 0x15, 0xc0, 0x6f, 0xbf, 0xa4, 0x69, 0x7d, 0xd1, 0x53, 0x0f, 0x0b, 0xc1, 0x2b, 0xad, 0x00, 0x44, 0x10, 0xe2, 0x9f, 0xb9, 0xf3, 0x0c, 0x98, 0x53, 0xf0, 0x60, 0xcb, 0xee, 0x7e, 0x5c, 0x83, 0x4a, 0xde, 0xa0, 0x7a, 0xcf, 0x50, 0x2b, 0x84, 0x09, 0xff, 0x42, 0xe4, 0x80, 0x2a, 0x64, 0x20, 0x9b, 0xb9, 0xba, 0xd4, 0x54, 0xca, 0xd8, 0xdc, 0x0a, 0x4d, 0xdd, 0x84, 0x91, 0x5e, 0x16, 0x90, 0x1d, 0xdc, 0xe3, 0x95, 0x55, 0xac, 0xf2, 0x8c, 0x9a, 0xcc, 0xb2, 0x6d, 0x17, 0x01, 0xe4, 0x01, 0xc6, 0xba, 0xe4, 0xb8, 0xd5, 0xbd, 0x7b, 0x43, 0xc9, 0x69, 0x6b, 0x40, 0xf7, 0xdc, 0x65, 0xa4, 0xf7, 0xca, 0x1f, 0xd8, 0xe5, 0xba, 0x4c, 0xdf, 0xe4, 0x64, 0x9e, 0x7d, 0xbd, 0x54, 0x13, 0x13,
		0xc6, 0x0c, 0xb8, 0x1d, 0x31, 0x0a, 0x49, 0xe2, 0x43, 0xb6, 0x95, 0x5f, 0x05, 0x6e, 0x66, 0xf4, 0x21, 0xa8, 0x65, 0xce, 0xf8, 0x8e, 0xcc, 0x16, 0x1e, 0xbb, 0xd8, 0x0e, 0xcb, 0xd2, 0x48, 0x37, 0xaf, 0x4e, 0x67, 0x45, 0xf1, 0x79, 0x69, 0xd2, 0xee, 0xa4, 0xb5, 0x01, 0xbf, 0x57, 0x0f, 0x68, 0x37, 0xbe, 0x4e, 0xff, 0xc9, 0xb9, 0x92, 0x23, 0x06, 0x75, 0xa0, 0x42, 0xe4, 0x0a, 0x30, 0xf0, 0xaf, 0xb0, 0x54, 0x88, 0x7c, 0xc0, 0xc1, 0x0c, 0x6d, 0x01, 0x36, 0x63, 0xf3, 0x3d, 0xbc, 0x72, 0xf6, 0x96, 0xc8, 0x87, 0xab, 0x8b, 0x0c, 0x91, 0x2f, 0x42, 0x2a, 0x11, 0xf6, 0x2d, 0x5e, 0x77, 0xce, 0x9c, 0xc1, 0x34, 0xe5, 0x2d, 0x9b, 0xd0, 0x37, 0x97, 0x0e, 0x39, 0xe5, 0xaa, 0xbe, 0x15, 0x3e, 0x6b, 0x1e, 0x73, 0xf6, 0xd7, 0xf4, 0xd6, 0x71, 0x70, 0xc6, 0xa1, 0xe6, 0x04, 0xd3, 0x7c, 0x2d, 0x1c, 0x98, 0x47, 0xdb, 0x8f, 0x59, 0x99, 0x2a, 0x57, 0x63, 0x14, 0xc7, 0x02, 0x42, 0x74, 0x57, 0x02, 0x22, 0xb2, 0x55, 0xe9, 0xf3, 0xe0, 0x76, 0x1c, 0x50, 0xbf, 0x43, 0x65, 0xbe, 0x52, 0xbd, 0x46, 0xf0, 0xfd, 0x5e, 0x25, 0xfe, 0x34, 0x50, 0x0d, 0x24, 0x7c, 0xfc, 0xfa, 0x82, 0x2f, 0x8c, 0x7d, 0x97, 0x1b, 0x07, 0x6b, 0x20, 0x6c, 0x9b, 0x7b, 0xae, 0xbf, 0xb3, 0x4f, 0x6e, 0xbb, 0xb6, 0xc4, 0xe9, 0xa5, 0x07, 0xa7, 0x74, 0x45, 0x16, 0x8a, 0x12, 0xee, 0x42, 0xc8, 0xea, 0xb5, 0x33, 0x69, 0xef, 0xff, 0x60, 0x6d, 0x99, 0xa3, 0x92, 0x5d, 0x0f, 0xbe, 0xb7, 0x4e, 0x1c, 0x85,
		0xef, 0x9e, 0x1d, 0x38, 0x72, 0x1f, 0xe0, 0xca, 0xc9, 0x90, 0x85, 0x3f, 0xa6, 0x5d, 0x60, 0x3f, 0xe6, 0x92, 0x08, 0x3b, 0xd4, 0xc3, 0xa2, 0x7e, 0x7c, 0x35, 0x49, 0xd4, 0x21, 0x38, 0x8c, 0x2c, 0x49, 0xb3, 0xcb, 0x33, 0xd4, 0xc2, 0x88, 0xdc, 0x09, 0xb3, 0x8a, 0x13, 0x95, 0x0f, 0xb4, 0x0a, 0xd1, 0x1d, 0xc8, 0xe4, 0x64, 0xb4, 0x24, 0x51, 0xe1, 0x0a, 0x22, 0xd4, 0x45, 0x77, 0x91, 0x0a, 0xc6, 0x61, 0xa1, 0x2c, 0x50, 0x84, 0x1c, 0x0c, 0xbe, 0x05, 0x1c, 0x3b, 0x4f, 0x27, 0x83, 0x33, 0xba, 0xfb, 0x7f, 0xa0, 0xc6, 0x38, 0xb4, 0x0c, 0x15, 0x49, 0x8f, 0xfa, 0x17, 0x76, 0xa9, 0x54, 0xf4, 0x6c, 0x7e, 0x5e, 0x39, 0xb8, 0xa8, 0x78, 0x86, 0x48, 0xb2, 0x18, 0xf1, 0xde, 0x0d, 0x24, 0xee, 0x6b, 0x01, 0x7d, 0x60, 0xfa, 0x35, 0xfe, 0x71, 0x0b, 0xfa, 0x8c, 0x79, 0x6c, 0x0b, 0x25, 0x84, 0x6d, 0x1a, 0x1d, 0xe0, 0x33, 0xa1, 0xa0, 0x8f, 0x47, 0x08, 0x4b, 0x5c, 0x8c, 0xc6, 0x1e, 0x2a, 0x6d, 0xd8, 0x3e, 0x09, 0x83, 0x96, 0xe6, 0xbc, 0x14, 0x55, 0x17, 0xcb, 0x50, 0x44, 0xdb, 0x80, 0xab, 0xb9, 0xf0, 0x1a, 0x3a, 0x9e, 0x23, 0xd5, 0x46, 0x73, 0x4b, 0xd0, 0x41, 0x9d, 0x29, 0x03, 0x59, 0x29, 0xeb, 0x82, 0x71, 0x09, 0x0c, 0x26, 0x10, 0x0f, 0x59, 0xd4, 0xd7, 0xb4, 0x4d, 0xe5, 0x35, 0xf5, 0x19, 0xef, 0xc7, 0xe7, 0x43, 0x0a, 0x3e, 0xeb, 0x3d, 0xc5, 0x55, 0xde, 0x04, 0xe7, 0x88, 0x72, 0x6c, 0xf7, 0x9d, 0x86, 0xb2, 0x0c, 0x83, 0x55, 0x20, 0x67, 0xc0, 0xc9, 0x15,
		0x3c, 0x76, 0x69, 0x80, 0x79, 0x68, 0x89, 0x16, 0x0a, 0xaf, 0xe4, 0x2c, 0xf0, 0x0e, 0x26, 0x74, 0x84, 0xfb, 0x27, 0xd4, 0x1c, 0x61, 0xbe, 0xe8, 0xc3, 0xce, 0x74, 0xd9, 0xf8, 0x5a, 0xa8, 0x63, 0x13, 0x27, 0xfa, 0xab, 0x93, 0x32, 0x25, 0x18, 0xb1, 0x78, 0x2f, 0xd3, 0x93, 0x0b, 0xc6, 0x5a, 0xda, 0xfe, 0xff, 0x7e, 0x38, 0x0c, 0x26, 0x44, 0x4c, 0x23, 0xe0, 0x8e, 0x64, 0xff, 0x07, 0xbc, 0x5b, 0x87, 0xd6, 0x3c, 0x8e, 0xe7, 0xd1, 0x78, 0x55, 0x00, 0x19, 0xbe, 0x98, 0x55, 0x1e, 0x16, 0xea, 0x63, 0x79, 0xb5, 0xaf, 0x9a, 0x20, 0x04, 0x8d, 0x3f, 0xdc, 0x15, 0x29, 0xc4, 0xe3, 0x9a, 0x82, 0x92, 0x85, 0xee, 0x1c, 0x37, 0xb3, 0xd7, 0xd2, 0x2e, 0x1e, 0xdb, 0x59, 0x87, 0xef, 0xa8, 0x9a, 0xaa, 0xa4, 0xed, 0x89, 0x33, 0xa8, 0xa7, 0x6c, 0x96, 0x9f, 0x26, 0xeb, 0xdc, 0x61, 0xc4, 0x8f, 0xd3, 0x2b, 0x81, 0x86, 0x6c, 0x9c, 0xc2, 0xb1, 0xb5, 0xbc, 0xa6, 0xd6, 0xd6, 0x1d, 0xce, 0x93, 0x78, 0xb3, 0xec, 0xa8, 0x64, 0x19, 0x13, 0x59, 0x1c, 0xb9, 0xbf, 0xd8, 0x7f, 0x27, 0x8e, 0x6f, 0x05, 0xd9, 0x1a, 0xa4, 0x1a, 0xc2, 0x46, 0x81, 0x52, 0xa5, 0xaf, 0x73, 0x35, 0x34, 0x88, 0x60, 0x46, 0x4d, 0x09, 0x87, 0xf1, 0x7e, 0x5e, 0xea, 0x32, 0x98, 0xb4, 0x68, 0x28, 0xff, 0x47, 0xde, 0x72, 0x9b, 0xc5, 0xfe, 0xb8, 0x93, 0xe8, 0x79, 0xe4, 0xa6, 0xd7, 0x63, 0x94, 0x29, 0x94, 0x33, 0x30, 0x61, 0xd4, 0x19, 0x36, 0x99, 0x94, 0x31, 0xbf, 0x93, 0x46, 0x04, 0xc0, 0xfe, 0x4d,
		0x92, 0xb4, 0xbc, 0xb2, 0x14, 0x3f, 0xf7, 0xce, 0x05, 0xcf, 0xf2, 0x5b, 0x66, 0xcb, 0x67, 0xa9, 0x8f, 0x63, 0xd4, 0x7c, 0x1d, 0x33, 0x6a, 0x05, 0xfb, 0xf7, 0x11, 0x03, 0x97, 0xff, 0x02, 0x1b, 0x6f, 0x15, 0x8b, 0x33, 0xe6, 0xf7, 0x5d, 0x93, 0x21, 0x9d, 0x17, 0xde, 0x9e, 0x87, 0xdc, 0xcd, 0x9a, 0x6a, 0x30, 0x3e, 0xa9, 0x70, 0xed, 0x93, 0x1d, 0x43, 0xb5, 0x5d, 0xb0, 0x46, 0x74, 0x73, 0x3b, 0x25, 0xfa, 0x0e, 0xe3, 0x70, 0x74, 0x2d, 0x75, 0xd6, 0x14, 0x67, 0x40, 0x31, 0xf9, 0x2c, 0xf6, 0x38, 0xea, 0x45, 0x33, 0xc1, 0xb6, 0xd5, 0x93, 0x0f, 0x5c, 0xaf, 0x3a, 0x53, 0x75, 0xd6, 0xe8, 0x97, 0xa0, 0x51, 0x3f, 0x96, 0x41, 0x32, 0x0b, 0x59, 0x48, 0xbf, 0x2b, 0x19, 0x67, 0x98, 0x42, 0xfe, 0x44, 0x23, 0x84, 0xa9, 0x09, 0x40, 0x4e, 0x10, 0x25, 0xdf, 0x68, 0x93, 0x6b, 0x0d, 0xa8, 0x51, 0x47, 0x55, 0xb7, 0xb8, 0x22, 0xab, 0xa3, 0x3c, 0x78, 0xd6, 0x8b, 0x4f, 0x2a, 0x73, 0xc1, 0x4a, 0x4a, 0xdd, 0x73, 0xb1, 0xc0, 0x8c, 0x5f, 0xf6, 0xe7, 0xbe, 0x9c, 0x96, 0xd6, 0x37, 0x91, 0x05, 0x52, 0xd1, 0x2f, 0xa9, 0xdc, 0xca, 0x11, 0x30, 0x6d, 0x4f, 0xb5, 0x6e, 0x39, 0x24, 0x28, 0x80, 0x54, 0x28, 0x87, 0xe6, 0x40, 0xeb, 0xd8, 0x7a, 0x1f, 0x63, 0x56, 0xc1, 0x4d, 0xa0, 0xf8,
	}

	// c0
	if 03 != c0c1[0] {
		t.Error("invalid")
	}

	// c1
	c1 := &chsC1S1{}
	// the schema of data must be schema0: key-digest.
	if err := c1.Parse(c0c1[1:1537], Schema0); err != nil {
		t.Error("invalid", err)
	}
	if c1.time != 0x000f64d0 {
		t.Error("invalid")
	}
	if c1.version != 0x80000702 {
		t.Error("invalid")
	}
	if ok, err := c1.Validate(); err != nil {
		t.Error("invalid", err)
	} else if !ok {
		t.Error("invalid")
	}
}
