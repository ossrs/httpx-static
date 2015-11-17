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
	"sync"
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

	// the input channel,
	// when got message from peer, write to this channel.
	// for example, c0c1 and c2 for server, or s0s1s2 for client.
	in chan []byte
	// the output channel,
	// when need to send to peer, write to this channel.
	// for example, c0c1 and c2 for client, or s0s1s2 for server.
	out chan []byte
}

func NewHsBytes() *hsBytes {
	return &hsBytes{
		c0c1c2: make([]byte, 3073),
		s0s1s2: make([]byte, 3073),
		// use buffer size 2 for we atmost got 2 messages to in/out.
		in:  make(chan []byte, 2),
		out: make(chan []byte, 2),
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

func (v *hsBytes) inCacheC0C1() (err error) {
	select {
	case v.in <- v.C0C1():
	default:
		return core.Overflow
	}

	core.Info.Println("cache c0c1 ok.")
	return
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

func (v *hsBytes) outCacheS0S1S2() (err error) {
	select {
	case v.out <- v.S0S1S2():
	default:
		return core.Overflow
	}

	core.Info.Println("cache s0s1s2 ok.")
	return
}

func (v *hsBytes) writeS0S1S2(w io.Writer) (err error) {
	r := bytes.NewReader(v.S0S1S2())
	if _, err = io.CopyN(w, r, 3073); err != nil {
		return
	}

	core.Info.Println("write s0s1s2 ok.")
	return
}

func (v *hsBytes) inCacheC2() (err error) {
	select {
	case v.in <- v.C2():
	default:
		return core.Overflow
	}

	core.Info.Println("cache c2 ok.")
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

// Rtmp message,
// which decode from RTMP chunked stream with raw body.
type RtmpMessage struct {
	// Four-byte field that contains a timestamp of the message.
	// The 4 bytes are packed in the big-endian order.
	// @remark, used as calc timestamp when decode and encode time.
	// @remark, we use 64bits for large time for jitter detect and hls.
	timestamp uint64
	// 4bytes.
	// Four-byte field that identifies the stream of the message. These
	// bytes are set in little-endian format.
	streamId uint32
	// 1byte.
	// One byte field to represent the message type. A range of type IDs
	// (1-7) are reserved for protocol control messages.
	messageType uint8
	// get the perfered cid(chunk stream id) which sendout over.
	// set at decoding, and canbe used for directly send message,
	// for example, dispatch to all connections.
	preferCid uint32
	// the payload of message, the SrsCommonMessage never know about the detail of payload,
	// user must use SrsProtocol.decode_message to get concrete packet.
	// @remark, not all message payload can be decoded to packet. for example,
	//       video/audio packet use raw bytes, no video/audio packet.
	payload []byte
}

func NewRtmpMessage() *RtmpMessage {
	return &RtmpMessage{
		payload: make([]byte, 0),
	}
}

// rtmp protocol stack.
type RtmpConnection struct {
	// to receive the quit message from server.
	wc core.WorkerContainer
	// the handshake bytes for RTMP.
	handshake *hsBytes
	// the underlayer transport.
	transport io.ReadWriteCloser
	// the RTMP protocol stack.
	stack *RtmpStack
	// input channel, receive message from client.
	in chan *RtmpMessage
	// output channel, to send to client.
	out chan *RtmpMessage
	// whether receiver and sender already quit.
	quit sync.WaitGroup
	// whether closed.
	closed bool
	lock   sync.Mutex
}

func NewRtmpConnection(transport io.ReadWriteCloser, wc core.WorkerContainer) *RtmpConnection {
	v := &RtmpConnection{
		wc:        wc,
		handshake: NewHsBytes(),
		transport: transport,
		stack:     NewRtmpStack(transport),
		in:        make(chan *RtmpMessage, 1),
		out:       make(chan *RtmpMessage, 1),
	}

	// start the receiver and sender.
	// directly use raw goroutine, for donot cause the container to quit.
	v.quit.Add(2)
	go core.Recover("rtmp receiver", v.receiver)
	go core.Recover("rtmp sender", v.sender)

	return v
}

// close the connection to client.
// TODO: FIXME: should be thread safe.
func (v *RtmpConnection) Close() {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.closed {
		return
	}

	// close transport,
	// to notify the wait goroutine to quit.
	if err := v.transport.Close(); err != nil {
		core.Warn.Println("ignore transport close err", err)
	}

	// close the out channel cache,
	// to notify the wait goroutine to quit.
	close(v.handshake.out)

	// try to read one to unblock the in channel.
	select {
	case <-v.in:
	default:
	}

	// close output to unblock the sender.
	close(v.out)

	// wait for sender and receiver to quit.
	v.quit.Wait()
	core.Warn.Println("rtmp receiver and sender quit.")

	return
}

func (v *RtmpConnection) Handshake() (err error) {
	// use short handshake timeout.
	timeout := 2100 * time.Millisecond

	// wait c0c1
	select {
	case <-v.handshake.in:
		// ok.
	case <-time.After(timeout):
		return core.Timeout
	case <-v.wc.QC():
		return v.wc.Quit()
	}

	// plain text required.
	if !v.handshake.ClientPlaintext() {
		return fmt.Errorf("only support rtmp plain text.")
	}

	// create s0s1s2 from c1.
	v.handshake.createS0S1S2()

	// cache the s0s1s2 for sender to write.
	if err = v.handshake.outCacheS0S1S2(); err != nil {
		return
	}

	// wait c2
	select {
	case <-v.handshake.in:
		// ok.
	case <-time.After(timeout):
		return core.Timeout
	case <-v.wc.QC():
		return v.wc.Quit()
	}

	return
}

func (v *RtmpConnection) ConnectApp() (r *RtmpRequest, err error) {
	r = &RtmpRequest{}

	// use longger connect timeout.
	timeout := 5000 * time.Millisecond

	// connect(tcUrl)
	select {
	case m := <-v.in:
		// ok.
		panic(m)
	case <-time.After(timeout):
		return nil, core.Timeout
	case <-v.wc.QC():
		return nil, v.wc.Quit()
	}

	// TODO: FIXME: implements it.
	return
}

func (v *RtmpConnection) receiver() (err error) {
	defer v.quit.Done()

	// read c0c2
	if err = v.handshake.readC0C1(v.transport); err != nil {
		return
	}

	if err = v.handshake.inCacheC0C1(); err != nil {
		return
	}

	// read c2
	if err = v.handshake.readC2(v.transport); err != nil {
		return
	}

	if err = v.handshake.inCacheC2(); err != nil {
		return
	}

	// message loop.
	for !v.closed {
		fn := func() (m *RtmpMessage, err error) {
			v.lock.Lock()
			defer v.lock.Unlock()

			if v.closed {
				core.Warn.Println("receiver break for closed.")
				return
			}

			if m, err = v.stack.ReadMessage(); err != nil {
				return
			}

			return
		}

		var m *RtmpMessage
		if m, err = fn(); err != nil {
			return
		}

		// cache the message.
		if m != nil {
			v.in <- m
		}
	}
	core.Warn.Println("receiver ok.")

	return
}

func (v *RtmpConnection) sender() (err error) {
	defer v.quit.Done()

	// write s0s1s2 to client.
	<-v.handshake.out
	if err = v.handshake.writeS0S1S2(v.transport); err != nil {
		return
	}

	// send out all messages in cache
	for m := range v.out {
		if err = v.stack.SendMessage(m); err != nil {
			return
		}
	}

	return
}

// RTMP protocol stack.
type RtmpStack struct {
	transport io.ReadWriter
}

func NewRtmpStack(transport io.ReadWriter) *RtmpStack {
	return &RtmpStack{
		transport: transport,
	}
}

func (v *RtmpStack) ReadMessage() (m *RtmpMessage, err error) {
	return
}

func (v *RtmpStack) SendMessage(m *RtmpMessage) (err error) {
	return
}

// 6.1.1. Chunk Basic Header
// The Chunk Basic Header encodes the chunk stream ID and the chunk
// type(represented by fmt field in the figure below). Chunk type
// determines the format of the encoded message header. Chunk Basic
// Header field may be 1, 2, or 3 bytes, depending on the chunk stream
// ID.
//
// The bits 0-5 (least significant) in the chunk basic header represent
// the chunk stream ID.
//
// Chunk stream IDs 2-63 can be encoded in the 1-byte version of this
// field.
//    0 1 2 3 4 5 6 7
//   +-+-+-+-+-+-+-+-+
//   |fmt|   cs id   |
//   +-+-+-+-+-+-+-+-+
//   Figure 6 Chunk basic header 1
//
// Chunk stream IDs 64-319 can be encoded in the 2-byte version of this
// field. ID is computed as (the second byte + 64).
//   0                   1
//   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   |fmt|    0      | cs id - 64    |
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   Figure 7 Chunk basic header 2
//
// Chunk stream IDs 64-65599 can be encoded in the 3-byte version of
// this field. ID is computed as ((the third byte)*256 + the second byte
// + 64).
//    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   |fmt|     1     |         cs id - 64            |
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   Figure 8 Chunk basic header 3
//
// cs id: 6 bits
// fmt: 2 bits
// cs id - 64: 8 or 16 bits
//
// Chunk stream IDs with values 64-319 could be represented by both 2-
// byte version and 3-byte version of this field.
func (v *RtmpStack) readBasicHeader() (fmt, cid uint8, err error) {
	return
}
