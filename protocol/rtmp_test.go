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
