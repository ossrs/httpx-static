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
	"encoding"
	"encoding/binary"
	"fmt"
	"github.com/ossrs/go-oryx/core"
	"io"
	"math"
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

// mix []byte and io.Reader
type mixReader struct {
	reader io.Reader
	next   io.Reader
}

func NewMixReader(a io.Reader, b io.Reader) io.Reader {
	return &mixReader{
		reader: a,
		next:   b,
	}
}

func (v *mixReader) Read(p []byte) (n int, err error) {
	for {
		if v.reader != nil {
			n, err = v.reader.Read(p)
			if err == io.EOF {
				v.reader = nil
				continue
			}
			return
		}

		if v.next != nil {
			n, err = v.next.Read(p)
			if err == io.EOF {
				v.next = nil
				continue
			}
			return
		}

		return 0, io.EOF
	}
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
		return core.OverflowError
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
		return core.OverflowError
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
		return core.OverflowError
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

// the rtmp client type.
type RtmpConnType uint8

const (
	Unknown RtmpConnType = iota
	Play
	FmlePublish
	FlashPublish
)

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
	// when receiver or sender quit, notify main goroutine.
	closing chan bool
	// whether closed.
	closed bool
	lock   sync.Mutex
}

func NewRtmpConnection(transport io.ReadWriteCloser, wc core.WorkerContainer) *RtmpConnection {
	v := &RtmpConnection{
		wc:        wc,
		handshake: NewHsBytes(),
		transport: transport,
		stack:     NewRtmpStack(transport, transport),
		in:        make(chan *RtmpMessage, 1),
		out:       make(chan *RtmpMessage, 1),
		closing:   make(chan bool, 2),
	}

	// wait for goroutine to run.
	wait := make(chan bool)

	// start the receiver and sender.
	// directly use raw goroutine, for donot cause the container to quit.
	v.quit.Add(2)
	go core.Recover("rtmp receiver", func() error {
		// noitfy the main goroutine to quit.
		defer func() {
			v.closing <- true
		}()

		// notify the main goroutine the receiver is ok.
		wait <- true

		if err := v.receiver(); err != nil {
			if v.stack.closing {
				return core.QuitError
			}
			return err
		}
		return nil
	})
	go core.Recover("rtmp sender", func() error {
		// noitfy the main goroutine to quit.
		defer func() {
			v.closing <- true
		}()

		// notify the main goroutine the sender is ok.
		wait <- true

		if err := v.sender(); err != nil {
			if v.stack.closing {
				return core.QuitError
			}
			return err
		}
		return nil
	})

	// wait for receiver and sender ok.
	<-wait
	<-wait

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

	// close the stack.
	v.stack.Close()

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
	core.Warn.Println("closed")

	return
}

// handshake with client, try complex then simple.
func (v *RtmpConnection) Handshake() (err error) {
	// use short handshake timeout.
	timeout := HandshakeTimeout

	// wait c0c1
	select {
	case <-v.handshake.in:
		// ok.
	case <-time.After(timeout):
		core.Error.Println("timeout for", timeout)
		return core.TimeoutError
	case <-v.closing:
		return core.ClosedError
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
		core.Error.Println("timeout for", timeout)
		return core.TimeoutError
	case <-v.closing:
		return core.ClosedError
	case <-v.wc.QC():
		return v.wc.Quit()
	}

	return
}

// do connect app with client, to discovery tcUrl.
func (v *RtmpConnection) ExpectConnectApp() (r *RtmpRequest, err error) {
	r = &RtmpRequest{}

	// use longger connect timeout.
	timeout := ConnectAppTimeout

	// connect(tcUrl)
	for {
		select {
		case m := <-v.in:
			var p RtmpPacket
			if p, err = v.stack.DecodeMessage(m); err != nil {
				return
			}
			if p, ok := p.(*RtmpConnectAppPacket); ok {
				if p, ok := p.CommandObject.Get("tcUrl").(*Amf0String); ok {
					r.TcUrl = string(*p)
				}
				core.Trace.Println("connect at", r.TcUrl)
			}
			return
		case <-time.After(timeout):
			core.Error.Println("timeout for", timeout)
			return nil, core.TimeoutError
		case <-v.closing:
			return nil, core.ClosedError
		case <-v.wc.QC():
			return nil, v.wc.Quit()
		}
	}

	return
}

// set ack size to client, client will send ack-size for each ack window
func (v *RtmpConnection) SetWindowAckSize(ack uint32) (err error) {
	// use longger service timeout.
	timeout := AckTimeout

	p := NewRtmpSetWindowAckSizePacket().(*RtmpSetWindowAckSizePacket)
	p.Ack = RtmpUint32(ack)

	var m *RtmpMessage
	if m, err = v.packet2Message(p, 0); err != nil {
		return
	}

	select {
	case v.out <- m:
		// ok
	case <-time.After(timeout):
		core.Error.Println("timeout for", timeout)
		return core.TimeoutError
	case <-v.closing:
		return core.ClosedError
	case <-v.wc.QC():
		return v.wc.Quit()
	}

	return
}

// @type: The sender can mark this message hard (0), soft (1), or dynamic (2)
// using the Limit type field.
func (v *RtmpConnection) SetPeerBandwidth(bw uint32, t uint8) (err error) {
	// use longger service timeout.
	timeout := SetPeerBandwidthTimeout

	p := NewRtmpSetPeerBandwidthPacket().(*RtmpSetPeerBandwidthPacket)
	p.Bandwidth = RtmpUint32(bw)
	p.Type = RtmpUint8(t)

	var m *RtmpMessage
	if m, err = v.packet2Message(p, 0); err != nil {
		return
	}

	select {
	case v.out <- m:
	// ok
	case <-time.After(timeout):
		core.Error.Println("timeout for", timeout)
		return core.TimeoutError
	case <-v.closing:
		return core.ClosedError
	case <-v.wc.QC():
		return v.wc.Quit()
	}

	return
}

// @param server_ip the ip of server.
func (v *RtmpConnection) ResponseConnectApp() (err error) {
	// use longger service timeout.
	timeout := ConnectAppTimeout

	p := NewRtmpConnectAppResPacket().(*RtmpConnectAppResPacket)

	p.Props.Set("fmsVer", NewAmf0String(fmt.Sprintf("FMS/%v", RtmpSigFmsVer)))
	p.Props.Set("capabilities", NewAmf0Number(127))
	p.Props.Set("mode", NewAmf0Number(1))

	p.Info.Set(StatusLevel, NewAmf0String(StatusLevelStatus))
	p.Info.Set(StatusCode, NewAmf0String(StatusCodeConnectSuccess))
	p.Info.Set(StatusDescription, NewAmf0String("Connection succeeded"))

	d := NewAmf0EcmaArray()
	p.Info.Set("data", d)

	d.Set("version", NewAmf0String(RtmpSigFmsVer))
	d.Set("oryx_sig", NewAmf0String(core.OryxSigKey))
	d.Set("oryx_server", NewAmf0String(core.OryxSigServer()))
	d.Set("oryx_role", NewAmf0String(core.OryxSigRole))
	d.Set("oryx_url", NewAmf0String(core.OryxSigUrl))
	d.Set("oryx_version", NewAmf0String(core.Version()))
	d.Set("oryx_site", NewAmf0String(core.OryxSigWeb))
	d.Set("oryx_email", NewAmf0String(core.OryxSigEmail))
	d.Set("oryx_copyright", NewAmf0String(core.OryxSigCopyright))
	d.Set("oryx_primary", NewAmf0String(core.OryxSigPrimary()))
	d.Set("oryx_authors", NewAmf0String(core.OryxSigAuthors))
	// for edge to directly get the id of client.
	// TODO: FIXME: support oryx_server_ip, oryx_pid, oryx_id

	var m *RtmpMessage
	if m, err = v.packet2Message(p, 0); err != nil {
		return
	}

	select {
	case v.out <- m:
	// ok
	case <-time.After(timeout):
		core.Error.Println("timeout for", timeout)
		return core.TimeoutError
	case <-v.closing:
		return core.ClosedError
	case <-v.wc.QC():
		return v.wc.Quit()
	}

	return
}

// response client the onBWDone message.
func (v *RtmpConnection) OnBwDone() (err error) {
	// use longger service timeout.
	timeout := OnBwDoneTimeout

	p := NewRtmpOnBwDonePacket().(*RtmpOnBwDonePacket)

	var m *RtmpMessage
	if m, err = v.packet2Message(p, 0); err != nil {
		return
	}

	select {
	case v.out <- m:
	// ok
	case <-time.After(timeout):
		core.Error.Println("timeout for", timeout)
		return core.TimeoutError
	case <-v.closing:
		return core.ClosedError
	case <-v.wc.QC():
		return v.wc.Quit()
	}

	return
}

// recv some message to identify the client.
// @stream_id, client will createStream to play or publish by flash,
//         the stream_id used to response the createStream request.
// @type, output the client type.
// @stream_name, output the client publish/play stream name. @see: SrsRequest.stream
// @duration, output the play client duration. @see: SrsRequest.duration
func (v *RtmpConnection) Identify(sid uint32) (connType RtmpConnType, streamName string, duration float64, err error) {
	// use longger connect timeout.
	timeout := IdentifyTimeout

	for {
		select {
		case m := <-v.in:
			var p RtmpPacket
			if p, err = v.stack.DecodeMessage(m); err != nil {
				return
			}

			switch mt := p.MessageType(); mt {
			// ignore silently.
			case RtmpMsgAcknowledgement, RtmpMsgSetChunkSize, RtmpMsgWindowAcknowledgementSize, RtmpMsgUserControlMessage:
				continue
			// matched
			case RtmpMsgAMF0CommandMessage, RtmpMsgAMF3CommandMessage:
				break
			// ignore with warning.
			default:
				core.Trace.Println("ignore rtmp message", mt)
				continue
			}

			switch p := p.(type) {
			case *RtmpCreateStreamPacket:
				return v.identifyCreateStream(sid, p)
			case *RtmpFMLEStartPacket:
				return v.identifyFmlePublish(sid, p)
			case *RtmpPlayPacket:
				return v.identifyPlay(sid, p)
			}
			// TODO: FIXME: implements it.

			// for other call msgs,
			// support response null first,
			// @see https://github.com/ossrs/srs/issues/106
			// TODO: FIXME: response in right way, or forward in edge mode.
			// TODO: FIXME: implements it.
		case <-time.After(timeout):
			core.Error.Println("timeout for", timeout)
			return 0, "", 0, core.TimeoutError
		case <-v.closing:
			return 0, "", 0, core.ClosedError
		case <-v.wc.QC():
			return 0, "", 0, v.wc.Quit()
		}
	}

	return
}
func (v *RtmpConnection) identifyCreateStream(sid uint32, p *RtmpCreateStreamPacket) (connType RtmpConnType, streamName string, duration float64, err error) {
	return
}
func (v *RtmpConnection) identifyFmlePublish(sid uint32, p *RtmpFMLEStartPacket) (connType RtmpConnType, streamName string, duration float64, err error) {
	return
}
func (v *RtmpConnection) identifyPlay(sid uint32, p *RtmpPlayPacket) (connType RtmpConnType, streamName string, duration float64, err error) {
	return
}

// parse the rtmp packet to message.
func (v *RtmpConnection) packet2Message(p RtmpPacket, sid uint32) (m *RtmpMessage, err error) {
	m = NewRtmpMessage()

	m.MessageType = p.MessageType()
	m.PreferCid = p.PreferCid()
	m.StreamId = sid

	if err = core.Marshal(p, &m.Payload); err != nil {
		return nil, err
	}

	return m, nil
}

// receiver goroutine.
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
		// got a message or error.
		var m *RtmpMessage
		if m, err = v.stack.ReadMessage(); err != nil {
			return
		}

		// check state.
		if func() bool {
			v.lock.Lock()
			defer v.lock.Unlock()

			return v.closed
		}() {
			break
		}

		// cache the message when got non empty one.
		if m != nil && m.Payload.Len() > 0 {
			v.in <- m
		}
	}
	core.Warn.Println("receiver ok.")

	return
}

// sender goroutine.
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

// 6.1.2. Chunk Message Header
// There are four different formats for the chunk message header,
// selected by the "fmt" field in the chunk basic header.
const (
	// 6.1.2.1. Type 0
	// Chunks of Type 0 are 11 bytes long. This type MUST be used at the
	// start of a chunk stream, and whenever the stream timestamp goes
	// backward (e.g., because of a backward seek).
	RtmpFmtType0 = iota
	// 6.1.2.2. Type 1
	// Chunks of Type 1 are 7 bytes long. The message stream ID is not
	// included; this chunk takes the same stream ID as the preceding chunk.
	// Streams with variable-sized messages (for example, many video
	// formats) SHOULD use this format for the first chunk of each new
	// message after the first.
	RtmpFmtType1
	// 6.1.2.3. Type 2
	// Chunks of Type 2 are 3 bytes long. Neither the stream ID nor the
	// message length is included; this chunk has the same stream ID and
	// message length as the preceding chunk. Streams with constant-sized
	// messages (for example, some audio and data formats) SHOULD use this
	// format for the first chunk of each message after the first.
	RtmpFmtType2
	// 6.1.2.4. Type 3
	// Chunks of Type 3 have no header. Stream ID, message length and
	// timestamp delta are not present; chunks of this type take values from
	// the preceding chunk. When a single message is split into chunks, all
	// chunks of a message except the first one, SHOULD use this type. Refer
	// to example 2 in section 6.2.2. Stream consisting of messages of
	// exactly the same size, stream ID and spacing in time SHOULD use this
	// type for all chunks after chunk of Type 2. Refer to example 1 in
	// section 6.2.1. If the delta between the first message and the second
	// message is same as the time stamp of first message, then chunk of
	// type 3 would immediately follow the chunk of type 0 as there is no
	// need for a chunk of type 2 to register the delta. If Type 3 chunk
	// follows a Type 0 chunk, then timestamp delta for this Type 3 chunk is
	// the same as the timestamp of Type 0 chunk.
	RtmpFmtType3
)

// the message type.
type RtmpMessageType uint8

func (v RtmpMessageType) String() string {
	switch v {
	case RtmpMsgSetChunkSize:
		return "SetChunkSize"
	case RtmpMsgAbortMessage:
		return "Abort"
	case RtmpMsgAcknowledgement:
		return "Acknowledgement"
	case RtmpMsgUserControlMessage:
		return "UserControl"
	case RtmpMsgWindowAcknowledgementSize:
		return "AcknowledgementSize"
	case RtmpMsgSetPeerBandwidth:
		return "SetPeerBandwidth"
	case RtmpMsgEdgeAndOriginServerCommand:
		return "EdgeOrigin"
	case RtmpMsgAMF3CommandMessage:
		return "Amf3Command"
	case RtmpMsgAMF0CommandMessage:
		return "Amf0Command"
	case RtmpMsgAMF0DataMessage:
		return "Amf0Data"
	case RtmpMsgAMF3DataMessage:
		return "Amf3Data"
	case RtmpMsgAMF3SharedObject:
		return "Amf3SharedObject"
	case RtmpMsgAMF0SharedObject:
		return "Amf0SharedObject"
	case RtmpMsgAudioMessage:
		return "Audio"
	case RtmpMsgVideoMessage:
		return "Video"
	case RtmpMsgAggregateMessage:
		return "Aggregate"
	default:
		return "unknown"
	}
}

func (v RtmpMessageType) isAudio() bool {
	return v == RtmpMsgAudioMessage
}

func (v RtmpMessageType) isVideo() bool {
	return v == RtmpMsgVideoMessage
}

func (v RtmpMessageType) isAmf0Command() bool {
	return v == RtmpMsgAMF0CommandMessage
}

func (v RtmpMessageType) isAmf0Data() bool {
	return v == RtmpMsgAMF0DataMessage
}

func (v RtmpMessageType) isAmf3Command() bool {
	return v == RtmpMsgAMF3CommandMessage
}
func (v RtmpMessageType) isAmf3Data() bool {
	return v == RtmpMsgAMF3DataMessage
}

func (v RtmpMessageType) isWindowAckledgementSize() bool {
	return v == RtmpMsgWindowAcknowledgementSize
}

func (v RtmpMessageType) isAckledgement() bool {
	return v == RtmpMsgAcknowledgement
}

func (v RtmpMessageType) isSetChunkSize() bool {
	return v == RtmpMsgSetChunkSize
}

func (v RtmpMessageType) isUserControlMessage() bool {
	return v == RtmpMsgUserControlMessage
}

func (v RtmpMessageType) isSetPeerBandwidth() bool {
	return v == RtmpMsgSetPeerBandwidth
}

func (v RtmpMessageType) isAggregate() bool {
	return v == RtmpMsgAggregateMessage
}

func (v RtmpMessageType) isAmf0() bool {
	return v.isAmf0Command() || v.isAmf0Data()
}

func (v RtmpMessageType) isAmf3() bool {
	return v.isAmf3Command() || v.isAmf3Data()
}

const (
	// 5. Protocol Control Messages
	// RTMP reserves message type IDs 1-7 for protocol control messages.
	// These messages contain information needed by the RTM Chunk Stream
	// protocol or RTMP itself. Protocol messages with IDs 1 & 2 are
	// reserved for usage with RTM Chunk Stream protocol. Protocol messages
	// with IDs 3-6 are reserved for usage of RTMP. Protocol message with ID
	// 7 is used between edge server and origin server.
	RtmpMsgSetChunkSize               RtmpMessageType = 0x01
	RtmpMsgAbortMessage               RtmpMessageType = 0x02
	RtmpMsgAcknowledgement            RtmpMessageType = 0x03
	RtmpMsgUserControlMessage         RtmpMessageType = 0x04
	RtmpMsgWindowAcknowledgementSize  RtmpMessageType = 0x05
	RtmpMsgSetPeerBandwidth           RtmpMessageType = 0x06
	RtmpMsgEdgeAndOriginServerCommand RtmpMessageType = 0x07
	// 3. Types of messages
	// The server and the client send messages over the network to
	// communicate with each other. The messages can be of any type which
	// includes audio messages, video messages, command messages, shared
	// object messages, data messages, and user control messages.
	// 3.1. Command message
	// Command messages carry the AMF-encoded commands between the client
	// and the server. These messages have been assigned message type value
	// of 20 for AMF0 encoding and message type value of 17 for AMF3
	// encoding. These messages are sent to perform some operations like
	// connect, createStream, publish, play, pause on the peer. Command
	// messages like onstatus, result etc. are used to inform the sender
	// about the status of the requested commands. A command message
	// consists of command name, transaction ID, and command object that
	// contains related parameters. A client or a server can request Remote
	// Procedure Calls (RPC) over streams that are communicated using the
	// command messages to the peer.
	RtmpMsgAMF3CommandMessage RtmpMessageType = 17 // 0x11
	RtmpMsgAMF0CommandMessage RtmpMessageType = 20 // 0x14
	// 3.2. Data message
	// The client or the server sends this message to send Metadata or any
	// user data to the peer. Metadata includes details about the
	// data(audio, video etc.) like creation time, duration, theme and so
	// on. These messages have been assigned message type value of 18 for
	// AMF0 and message type value of 15 for AMF3.
	RtmpMsgAMF0DataMessage RtmpMessageType = 18 // 0x12
	RtmpMsgAMF3DataMessage RtmpMessageType = 15 // 0x0F
	// 3.3. Shared object message
	// A shared object is a Flash object (a collection of name value pairs)
	// that are in synchronization across multiple clients, instances, and
	// so on. The message types kMsgContainer=19 for AMF0 and
	// kMsgContainerEx=16 for AMF3 are reserved for shared object events.
	// Each message can contain multiple events.
	RtmpMsgAMF3SharedObject RtmpMessageType = 16 // 0x10
	RtmpMsgAMF0SharedObject RtmpMessageType = 19 // 0x13
	// 3.4. Audio message
	// The client or the server sends this message to send audio data to the
	// peer. The message type value of 8 is reserved for audio messages.
	RtmpMsgAudioMessage RtmpMessageType = 8 // 0x08
	// 3.5. Video message
	// The client or the server sends this message to send video data to the
	// peer. The message type value of 9 is reserved for video messages.
	// These messages are large and can delay the sending of other type of
	// messages. To avoid such a situation, the video message is assigned
	// the lowest priority.
	RtmpMsgVideoMessage RtmpMessageType = 9 // 0x09
	// 3.6. Aggregate message
	// An aggregate message is a single message that contains a list of submessages.
	// The message type value of 22 is reserved for aggregate
	// messages.
	RtmpMsgAggregateMessage RtmpMessageType = 22 // 0x16
)

const (
	// the chunk stream id used for some under-layer message,
	// for example, the PC(protocol control) message.
	RtmpCidProtocolControl = 0x02 + iota
	// the AMF0/AMF3 command message, invoke method and return the result, over NetConnection.
	// generally use 0x03.
	RtmpCidOverConnection
	// the AMF0/AMF3 command message, invoke method and return the result, over NetConnection,
	// the midst state(we guess).
	// rarely used, e.g. onStatus(NetStream.Play.Reset).
	RtmpCidOverConnection2
	// the stream message(amf0/amf3), over NetStream.
	// generally use 0x05.
	RtmpCidOverStream
	// the stream message(amf0/amf3), over NetStream, the midst state(we guess).
	// rarely used, e.g. play("mp4:mystram.f4v")
	RtmpCidOverStream2
	// the stream message(video), over NetStream
	// generally use 0x06.
	RtmpCidVideo
	// the stream message(audio), over NetStream.
	// generally use 0x07.
	RtmpCidAudio
)

// 6.1. Chunk Format
// Extended timestamp: 0 or 4 bytes
// This field MUST be sent when the normal timsestamp is set to
// 0xffffff, it MUST NOT be sent if the normal timestamp is set to
// anything else. So for values less than 0xffffff the normal
// timestamp field SHOULD be used in which case the extended timestamp
// MUST NOT be present. For values greater than or equal to 0xffffff
// the normal timestamp field MUST NOT be used and MUST be set to
// 0xffffff and the extended timestamp MUST be sent.
const RtmpExtendedTimestamp = 0xFFFFFF

// the default chunk size for system.
const RtmpServerChunkSize = 60000

// 6. Chunking, RTMP protocol default chunk size.
const RtmpProtocolChunkSize = 128

// 6. Chunking
// The chunk size is configurable. It can be set using a control
// message(Set Chunk Size) as described in section 7.1. The maximum
// chunk size can be 65536 bytes and minimum 128 bytes. Larger values
// reduce CPU usage, but also commit to larger writes that can delay
// other content on lower bandwidth connections. Smaller chunks are not
// good for high-bit rate streaming. Chunk size is maintained
// independently for each direction.
const RtmpMinChunkSize = 128
const RtmpMaxChunkSize = 65536

const (
	// amf0 command message, command name macros
	Amf0CommandConnect       = "connect"
	Amf0CommandCreateStream  = "createStream"
	Amf0CommandCloseStream   = "closeStream"
	Amf0CommandPlay          = "play"
	Amf0CommandPause         = "pause"
	Amf0CommandOnBwDone      = "onBWDone"
	Amf0CommandOnStatus      = "onStatus"
	Amf0CommandResult        = "_result"
	Amf0CommandError         = "_error"
	Amf0CommandReleaseStream = "releaseStream"
	Amf0CommandFcPublish     = "FCPublish"
	Amf0CommandUnpublish     = "FCUnpublish"
	Amf0CommandPublish       = "publish"
	Amf0DataSampleAccess     = "|RtmpSampleAccess"

	// the signature for packets to client.
	RtmpSigFmsVer   = "3,5,3,888"
	RtmpSigAmf0Ver  = 0
	RtmpSigClientId = "ASAICiss"

	// onStatus consts.
	StatusLevel       = "level"
	StatusCode        = "code"
	StatusDescription = "description"
	StatusDetails     = "details"
	StatusClientId    = "clientid"
	// status value
	StatusLevelStatus = "status"
	// status error
	StatusLevelError = "error"
	// code value
	StatusCodeConnectSuccess   = "NetConnection.Connect.Success"
	StatusCodeConnectRejected  = "NetConnection.Connect.Rejected"
	StatusCodeStreamReset      = "NetStream.Play.Reset"
	StatusCodeStreamStart      = "NetStream.Play.Start"
	StatusCodeStreamPause      = "NetStream.Pause.Notify"
	StatusCodeStreamUnpause    = "NetStream.Unpause.Notify"
	StatusCodePublishStart     = "NetStream.Publish.Start"
	StatusCodeDataStart        = "NetStream.Data.Start"
	StatusCodeUnpublishSuccess = "NetStream.Unpublish.Success"
)

// the uint8 which suppport marshal and unmarshal.
type RtmpUint8 uint8

func (v *RtmpUint8) MarshalBinary() (data []byte, err error) {
	return []byte{byte(*v)}, nil
}

func (v *RtmpUint8) Size() int {
	return 1
}

func (v *RtmpUint8) UnmarshalBinary(data []byte) (err error) {
	if len(data) == 0 {
		return io.EOF
	}
	*v = RtmpUint8(data[0])
	return
}

// the uint16 which suppport marshal and unmarshal.
type RtmpUint16 uint16

func (v *RtmpUint16) MarshalBinary() (data []byte, err error) {
	return []byte{byte(*v >> 8), byte(*v)}, nil
}

func (v *RtmpUint16) Size() int {
	return 2
}

func (v *RtmpUint16) UnmarshalBinary(data []byte) (err error) {
	if len(data) < 2 {
		return io.EOF
	}
	*v = RtmpUint16(uint16(data[0]) | uint16(data[1])<<8)
	return
}

// the uint32 which suppport marshal and unmarshal.
type RtmpUint32 uint32

func (v *RtmpUint32) MarshalBinary() (data []byte, err error) {
	return []byte{byte(*v >> 24), byte(*v >> 16), byte(*v >> 8), byte(*v)}, nil
}

func (v *RtmpUint32) Size() int {
	return 4
}

func (v *RtmpUint32) UnmarshalBinary(data []byte) (err error) {
	if len(data) < 4 {
		return io.EOF
	}
	*v = RtmpUint32(uint32(data[3]) | uint32(data[2])<<8 | uint32(data[1])<<16 | uint32(data[0])<<24)
	return
}

// Rtmp message,
// which decode from RTMP chunked stream with raw body.
type RtmpMessage struct {
	// Four-byte field that contains a timestamp of the message.
	// The 4 bytes are packed in the big-endian order.
	// @remark, used as calc timestamp when decode and encode time.
	// @remark, we use 64bits for large time for jitter detect and hls.
	Timestamp uint64
	// 4bytes.
	// Four-byte field that identifies the stream of the message. These
	// bytes are set in little-endian format.
	StreamId uint32
	// 1byte.
	// One byte field to represent the message type. A range of type IDs
	// (1-7) are reserved for protocol control messages.
	MessageType RtmpMessageType
	// get the perfered cid(chunk stream id) which sendout over.
	// set at decoding, and canbe used for directly send message,
	// for example, dispatch to all connections.
	PreferCid uint32
	// the payload of message, the SrsCommonMessage never know about the detail of payload,
	// user must use SrsProtocol.decode_message to get concrete packet.
	// @remark, not all message payload can be decoded to packet. for example,
	//       video/audio packet use raw bytes, no video/audio packet.
	Payload bytes.Buffer
}

func NewRtmpMessage() *RtmpMessage {
	return &RtmpMessage{}
}

// RTMP packet, which can be
// decode from and encode to message payload.
type RtmpPacket interface {
	// all packet can marshaler and unmarshaler.
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	// the cid(chunk id) specifies the chunk to send data over.
	// generally, each message perfer some cid, for example,
	// all protocol control messages perfer RTMP_CID_ProtocolControl,
	// SrsSetWindowAckSizePacket is protocol control message.
	PreferCid() uint32
	// subpacket must override to provide the right message type.
	// the message type set the RTMP message type in header.
	MessageType() RtmpMessageType
}

// 4.1.1. connect
// The client sends the connect command to the server to request
// connection to a server application instance.
type RtmpConnectAppPacket struct {
	// Name of the command. Set to "connect".
	Name Amf0String
	// Always set to 1.
	TransactionId Amf0Number
	// Command information object which has the name-value pairs.
	// @remark: alloc in packet constructor, user can directly use it,
	//       user should never alloc it again which will cause memory leak.
	// @remark, never be NULL.
	CommandObject *Amf0Object
	// Any optional information
	// @remark, optional, init to and maybe NULL.
	Args *Amf0Object
}

func NewRtmpConnectAppPacket() RtmpPacket {
	return &RtmpConnectAppPacket{
		Name:          Amf0String(Amf0CommandConnect),
		TransactionId: Amf0Number(1.0),
		CommandObject: NewAmf0Object(),
		Args:          nil,
	}
}

func (v *RtmpConnectAppPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, v.CommandObject, v.Args)
}

func (v *RtmpConnectAppPacket) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)
	if err = core.Unmarshals(b, &v.Name, &v.TransactionId, v.CommandObject); err != nil {
		return
	}

	if b.Len() > 0 {
		v.Args = NewAmf0Object()
		return core.Unmarshals(b, v.Args)
	}

	return
}

func (v *RtmpConnectAppPacket) PreferCid() uint32 {
	return RtmpCidOverConnection
}

func (v *RtmpConnectAppPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// response for SrsConnectAppPacket.
type RtmpConnectAppResPacket struct {
	// _result or _error; indicates whether the response is result or error.
	Name Amf0String
	// Transaction ID is 1 for call connect responses
	TransactionId Amf0Number
	// Name-value pairs that describe the properties(fmsver etc.) of the connection.
	// @remark, never be NULL.
	Props *Amf0Object
	// Name-value pairs that describe the response from|the server. 'code',
	// 'level', 'description' are names of few among such information.
	// @remark, never be NULL.
	Info *Amf0Object
}

func NewRtmpConnectAppResPacket() RtmpPacket {
	return &RtmpConnectAppResPacket{
		Name:          Amf0String(Amf0CommandResult),
		TransactionId: Amf0Number(1.0),
		Props:         NewAmf0Object(),
		Info:          NewAmf0Object(),
	}
}

func (v *RtmpConnectAppResPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, v.Props, v.Info)
}

func (v *RtmpConnectAppResPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Name, &v.TransactionId, v.Props, v.Info)
}

func (v *RtmpConnectAppResPacket) PreferCid() uint32 {
	return RtmpCidOverConnection
}

func (v *RtmpConnectAppResPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// 4.1.3. createStream
// The client sends this command to the server to create a logical
// channel for message communication The publishing of audio, video, and
// metadata is carried out over stream channel created using the
// createStream command.
type RtmpCreateStreamPacket struct {
	// Name of the command. Set to "createStream".
	Name Amf0String
	// Transaction ID of the command.
	TransactionId Amf0Number
	// If there exists any command info this is set, else this is set to null type.
	// @remark, never be NULL, an AMF0 null instance.
	Command Amf0Null
}

func NewRtmpCreateStreamPacket() RtmpPacket {
	return &RtmpCreateStreamPacket{
		Name: Amf0String(Amf0CommandCreateStream),
	}
}

func (v *RtmpCreateStreamPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, &v.Command)
}

func (v *RtmpCreateStreamPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Name, &v.TransactionId, &v.Command)
}

func (v *RtmpCreateStreamPacket) PreferCid() uint32 {
	return RtmpCidProtocolControl
}

func (v *RtmpCreateStreamPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// 5.5. Window Acknowledgement Size (5)
// The client or the server sends this message to inform the peer which
// window size to use when sending acknowledgment.
type RtmpSetWindowAckSizePacket struct {
	Ack RtmpUint32
}

func NewRtmpSetWindowAckSizePacket() RtmpPacket {
	return &RtmpSetWindowAckSizePacket{}
}

func (v *RtmpSetWindowAckSizePacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Ack)
}

func (v *RtmpSetWindowAckSizePacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Ack)
}

func (v *RtmpSetWindowAckSizePacket) PreferCid() uint32 {
	return RtmpCidProtocolControl
}

func (v *RtmpSetWindowAckSizePacket) MessageType() RtmpMessageType {
	return RtmpMsgWindowAcknowledgementSize
}

// 5.6. Set Peer Bandwidth (6)
type RtmpSetPeerBandwidthType uint8

const (
	// The sender can mark this message hard (0), soft (1), or dynamic (2)
	// using the Limit type field.
	Hard RtmpSetPeerBandwidthType = iota
	Soft
	Dynamic
)

// 5.6. Set Peer Bandwidth (6)
// The client or the server sends this message to update the output
// bandwidth of the peer.
type RtmpSetPeerBandwidthPacket struct {
	Bandwidth RtmpUint32
	// @see RtmpSetPeerBandwidthType
	Type RtmpUint8
}

func NewRtmpSetPeerBandwidthPacket() RtmpPacket {
	return &RtmpSetPeerBandwidthPacket{
		Type: RtmpUint8(Dynamic),
	}
}

func (v *RtmpSetPeerBandwidthPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Bandwidth, &v.Type)
}

func (v *RtmpSetPeerBandwidthPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Bandwidth, &v.Type)
}

func (v *RtmpSetPeerBandwidthPacket) PreferCid() uint32 {
	return RtmpCidProtocolControl
}

func (v *RtmpSetPeerBandwidthPacket) MessageType() RtmpMessageType {
	return RtmpMsgSetPeerBandwidth
}

// when bandwidth test done, notice client.
type RtmpOnBwDonePacket struct {
	// Name of command. Set to "onBWDone"
	Name Amf0String
	// Transaction ID set to 0.
	TransactionId Amf0Number
	// Command information does not exist. Set to null type.
	// @remark, never be NULL, an AMF0 null instance.
	Args Amf0Null
}

func NewRtmpOnBwDonePacket() RtmpPacket {
	return &RtmpOnBwDonePacket{
		Name: Amf0String(Amf0CommandOnBwDone),
	}
}

func (v *RtmpOnBwDonePacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, &v.Args)
}

func (v *RtmpOnBwDonePacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Name, &v.TransactionId, &v.Args)
}

func (v *RtmpOnBwDonePacket) PreferCid() uint32 {
	return RtmpCidOverConnection
}

func (v *RtmpOnBwDonePacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// FMLE start publish: ReleaseStream/PublishStream
type RtmpFMLEStartPacket struct {
	// Name of the command
	Name Amf0String
	// the transaction ID to get the response.
	TransactionId Amf0Number
	// If there exists any command info this is set, else this is set to null type.
	// @remark, never be NULL, an AMF0 null instance.
	Command Amf0Null
	// the stream name to start publish or release.
	StreamName Amf0String
}

func NewRtmpFMLEStartPacket() RtmpPacket {
	return &RtmpFMLEStartPacket{
		Name: Amf0String(Amf0CommandReleaseStream),
	}
}

func (v *RtmpFMLEStartPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, &v.Command, &v.StreamName)
}

func (v *RtmpFMLEStartPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Name, &v.TransactionId, &v.Command, &v.StreamName)
}

func (v *RtmpFMLEStartPacket) PreferCid() uint32 {
	return RtmpCidOverConnection
}

func (v *RtmpFMLEStartPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// 4.2.1. play
// The client sends this command to the server to play a stream.
type RtmpPlayPacket struct {
	// Name of the command. Set to "play".
	Name Amf0String
	// Transaction ID set to 0.
	TransactionId Amf0Number
	// Command information does not exist. Set to null type.
	// @remark, never be NULL, an AMF0 null instance.
	Command Amf0Null
	// Name of the stream to play.
	// To play video (FLV) files, specify the name of the stream without a file
	//       extension (for example, "sample").
	// To play back MP3 or ID3 tags, you must precede the stream name with mp3:
	//       (for example, "mp3:sample".)
	// To play H.264/AAC files, you must precede the stream name with mp4: and specify the
	//       file extension. For example, to play the file sample.m4v, specify
	//       "mp4:sample.m4v"
	Stream Amf0String
	// An optional parameter that specifies the start time in seconds.
	// The default value is -2, which means the subscriber first tries to play the live
	//       stream specified in the Stream Name field. If a live stream of that name is
	//       not found, it plays the recorded stream specified in the Stream Name field.
	// If you pass -1 in the Start field, only the live stream specified in the Stream
	//       Name field is played.
	// If you pass 0 or a positive number in the Start field, a recorded stream specified
	//       in the Stream Name field is played beginning from the time specified in the
	//       Start field.
	// If no recorded stream is found, the next item in the playlist is played.
	Start *Amf0Number
	// An optional parameter that specifies the duration of playback in seconds.
	// The default value is -1. The -1 value means a live stream is played until it is no
	//       longer available or a recorded stream is played until it ends.
	// If u pass 0, it plays the single frame since the time specified in the Start field
	//       from the beginning of a recorded stream. It is assumed that the value specified
	//       in the Start field is equal to or greater than 0.
	// If you pass a positive number, it plays a live stream for the time period specified
	//       in the Duration field. After that it becomes available or plays a recorded
	//       stream for the time specified in the Duration field. (If a stream ends before the
	//       time specified in the Duration field, playback ends when the stream ends.)
	// If you pass a negative number other than -1 in the Duration field, it interprets the
	//       value as if it were -1.
	Duration *Amf0Number
	// An optional Boolean value or number that specifies whether to flush any
	// previous playlist.
	Reset *Amf0Boolean
}

func NewRtmpPlayPacket() RtmpPacket {
	return &RtmpPlayPacket{
		Name: Amf0String(Amf0CommandPlay),
	}
}

func (v *RtmpPlayPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, &v.Command, &v.Stream, v.Start, v.Duration, v.Reset)
}

func (v *RtmpPlayPacket) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)
	if err = core.Unmarshals(b, &v.Name, &v.TransactionId, &v.Command, &v.Stream); err != nil {
		return
	}

	if b.Len() > 0 {
		v.Start = NewAmf0Number(0)
		if err = core.Unmarshals(b, v.Start); err != nil {
			return
		}
	}
	if b.Len() > 0 {
		v.Duration = NewAmf0Number(0)
		if err = core.Unmarshals(b, v.Duration); err != nil {
			return
		}
	}
	if b.Len() > 0 {
		v.Reset = NewAmf0Bool(false)
		if err = core.Unmarshals(b, v.Reset); err != nil {
			return
		}
	}

	return
}

func (v *RtmpPlayPacket) PreferCid() uint32 {
	return RtmpCidOverStream
}

func (v *RtmpPlayPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// 7.1. Set Chunk Size
// Protocol control message 1, Set Chunk Size, is used to notify the
// peer about the new maximum chunk size.
type RtmpSetChunkSizePacket struct {
	ChunkSize RtmpUint32
}

func NewRtmpSetChunkSizePacket() RtmpPacket {
	return &RtmpSetChunkSizePacket{}
}

func (v *RtmpSetChunkSizePacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.ChunkSize)
}

func (v *RtmpSetChunkSizePacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.ChunkSize)
}

func (v *RtmpSetChunkSizePacket) PreferCid() uint32 {
	return RtmpCidProtocolControl
}

func (v *RtmpSetChunkSizePacket) MessageType() RtmpMessageType {
	return RtmpMsgSetChunkSize
}

// the empty packet is a sample rtmp packet.
type RtmpEmptyPacket struct {
	Id Amf0Number
}

func NewRtmpEmptyPacket() RtmpPacket {
	return &RtmpEmptyPacket{}
}

func (v *RtmpEmptyPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Id)
}

func (v *RtmpEmptyPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Id)
}

func (v *RtmpEmptyPacket) PreferCid() uint32 {
	return RtmpCidOverConnection
}

func (v *RtmpEmptyPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// 3.7. User Control message
type RtmpPcucEventType RtmpUint16

const (
	// 2bytes event-type and generally, 4bytes event-data

	// The server sends this event to notify the client
	// that a stream has become functional and can be
	// used for communication. By default, this event
	// is sent on ID 0 after the application connect
	// command is successfully received from the
	// client. The event data is 4-byte and represents
	// the stream ID of the stream that became
	// functional.
	RtmpPcucStreamBegin RtmpPcucEventType = 0x00

	// The server sends this event to notify the client
	// that the playback of data is over as requested
	// on this stream. No more data is sent without
	// issuing additional commands. The client discards
	// the messages received for the stream. The
	// 4 bytes of event data represent the ID of the
	// stream on which playback has ended.
	RtmpPcucStreamEOF RtmpPcucEventType = 0x01

	// The server sends this event to notify the client
	// that there is no more data on the stream. If the
	// server does not detect any message for a time
	// period, it can notify the subscribed clients
	// that the stream is dry. The 4 bytes of event
	// data represent the stream ID of the dry stream.
	RtmpPcucStreamDry RtmpPcucEventType = 0x02

	// The client sends this event to inform the server
	// of the buffer size (in milliseconds) that is
	// used to buffer any data coming over a stream.
	// This event is sent before the server starts
	// processing the stream. The first 4 bytes of the
	// event data represent the stream ID and the next
	// 4 bytes represent the buffer length, in
	// milliseconds.
	RtmpPcucSetBufferLength RtmpPcucEventType = 0x03 // 8bytes event-data

	// The server sends this event to notify the client
	// that the stream is a recorded stream. The
	// 4 bytes event data represent the stream ID of
	// the recorded stream.
	RtmpPcucStreamIsRecorded RtmpPcucEventType = 0x04

	// The server sends this event to test whether the
	// client is reachable. Event data is a 4-byte
	// timestamp, representing the local server time
	// when the server dispatched the command. The
	// client responds with kMsgPingResponse on
	// receiving kMsgPingRequest.
	RtmpPcucPingRequest RtmpPcucEventType = 0x06

	// The client sends this event to the server in
	// response to the ping request. The event data is
	// a 4-byte timestamp, which was received with the
	// kMsgPingRequest request.
	RtmpPcucPingResponse RtmpPcucEventType = 0x07

	// for PCUC size=3, the payload is "00 1A 01",
	// where we think the event is 0x001a, fms defined msg,
	// which has only 1bytes event data.
	RtmpPcucFmsEvent0 RtmpPcucEventType = 0x1a
)

// 5.4. User Control Message (4)
//
// for the EventData is 4bytes.
// Stream Begin(=0)              4-bytes stream ID
// Stream EOF(=1)                4-bytes stream ID
// StreamDry(=2)                 4-bytes stream ID
// SetBufferLength(=3)           8-bytes 4bytes stream ID, 4bytes buffer length.
// StreamIsRecorded(=4)          4-bytes stream ID
// PingRequest(=6)               4-bytes timestamp local server time
// PingResponse(=7)              4-bytes timestamp received ping request.
//
// 3.7. User Control message
// +------------------------------+-------------------------
// | Event Type ( 2- bytes ) | Event Data
// +------------------------------+-------------------------
// Figure 5 Pay load for the 'User Control Message'.
type RtmpUserControlPacket struct {
	// Event type is followed by Event data.
	// @see RtmpPcucEventType
	EventType RtmpUint16
	// the event data generally in 4bytes.
	// @remark for event type is 0x001a, only 1bytes.
	EventData RtmpUint32
	// 4bytes if event_type is RtmpPcucSetBufferLength; otherwise 0.
	ExtraData RtmpUint32
}

func NewRtmpUserControlPacket() RtmpPacket {
	return &RtmpUserControlPacket{
		EventType: RtmpUint16(RtmpPcucStreamBegin),
	}
}

func (v *RtmpUserControlPacket) MarshalBinary() (data []byte, err error) {
	if RtmpPcucEventType(v.EventType) == RtmpPcucSetBufferLength {
		return core.Marshals(&v.EventType, &v.EventData, &v.ExtraData)
	}
	return core.Marshals(&v.EventType, &v.EventData)
}

func (v *RtmpUserControlPacket) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)
	if err = core.Unmarshals(b, &v.EventType, &v.EventData); err != nil {
		return
	}
	if RtmpPcucEventType(v.EventType) == RtmpPcucSetBufferLength {
		return core.Unmarshals(b, &v.ExtraData)
	}
	return
}

func (v *RtmpUserControlPacket) PreferCid() uint32 {
	return RtmpCidProtocolControl
}

func (v *RtmpUserControlPacket) MessageType() RtmpMessageType {
	return RtmpMsgUserControlMessage
}

// incoming chunk stream maybe interlaced,
// use the chunk stream to cache the input RTMP chunk streams.
type RtmpChunk struct {
	// the fmt of basic header.
	fmt uint8
	// the cid of basic header.
	cid uint32
	// the calculated timestamp.
	timestamp uint64
	// whether this chunk stream has extended timestamp.
	hasExtendedTimestamp bool
	// whether this chunk stream is fresh.
	isFresh bool
	// the partial message which not completed.
	partialMessage *RtmpMessage

	// 4.1. Message Header
	// 3bytes.
	// Three-byte field that contains a timestamp delta of the message.
	// @remark, only used for decoding message from chunk stream.
	timestampDelta uint32
	// 3bytes.
	// Three-byte field that represents the size of the payload in bytes.
	// It is set in big-endian format.
	payloadLength uint32
	// 1byte.
	// One byte field to represent the message type. A range of type IDs
	// (1-7) are reserved for protocol control messages.
	messageType uint8
	// 4bytes.
	// Four-byte field that identifies the stream of the message. These
	// bytes are set in little-endian format.
	streamId uint32
}

func NewRtmpChunk(cid uint32) *RtmpChunk {
	return &RtmpChunk{
		cid:     cid,
		isFresh: true,
	}
}

// RTMP protocol stack.
type RtmpStack struct {
	// the input and output stream.
	in  io.Reader
	out io.Writer
	// use bytes.Buffer to parse RTMP.
	inb bytes.Buffer
	// the chunks for RTMP,
	// key is the cid from basic header.
	chunks map[uint32]*RtmpChunk
	// input chunk size, default to 128, set by peer packet.
	inChunkSize uint32
	// output chunk size, default to 128, set by peer packet.
	outChunkSize uint32
	// whether the stack is closing.
	closing bool
}

func NewRtmpStack(r io.Reader, w io.Writer) *RtmpStack {
	return &RtmpStack{
		in:           r,
		out:          w,
		chunks:       make(map[uint32]*RtmpChunk),
		inChunkSize:  RtmpProtocolChunkSize,
		outChunkSize: RtmpProtocolChunkSize,
	}
}

func (v *RtmpStack) Close() {
	v.closing = true
}

func (v *RtmpStack) DecodeMessage(m *RtmpMessage) (p RtmpPacket, err error) {
	b := bytes.NewBuffer(m.Payload.Bytes())

	// decode specified packet type
	if m.MessageType.isAmf0() || m.MessageType.isAmf3() {
		// skip 1bytes to decode the amf3 command.
		if m.MessageType.isAmf3() && b.Len() > 0 {
			b.ReadByte()
		}

		// amf0 command message.
		// need to read the command name.
		var c Amf0String
		if err = c.UnmarshalBinary(b.Bytes()); err != nil {
			return
		}

		// result/error packet
		if c == Amf0CommandResult || c == Amf0CommandError {
			// TODO: FIXME: implements it.
		}

		// decode command object.
		switch c {
		case Amf0CommandConnect:
			p = NewRtmpConnectAppPacket()
		case Amf0CommandCreateStream:
			p = NewRtmpCreateStreamPacket()
		case Amf0CommandPlay:
			p = NewRtmpPlayPacket()
		case Amf0CommandPause:
			// TODO: FIXME: implements it.
		case Amf0CommandReleaseStream, Amf0CommandFcPublish, Amf0CommandUnpublish:
			p = NewRtmpFMLEStartPacket()
		case Amf0CommandPublish:
			// TODO: FIXME: implements it.
		// TODO: FIXME: implements it.
		default:
			core.Info.Println("drop command message, name is", c)
		}
	} else if m.MessageType.isUserControlMessage() {
		p = NewRtmpUserControlPacket()
	} else if m.MessageType.isWindowAckledgementSize() {
		p = NewRtmpSetWindowAckSizePacket()
	} else if m.MessageType.isSetChunkSize() {
		p = NewRtmpSetChunkSizePacket()
	} else {
		if !m.MessageType.isSetPeerBandwidth() && !m.MessageType.isAckledgement() {
			core.Trace.Println("drop unknown message, type is", m.MessageType)
		}
	}

	// unmarshal the discoveried packet.
	if p != nil {
		if err = p.UnmarshalBinary(b.Bytes()); err != nil {
			return
		}
	}

	return
}

func (v *RtmpStack) ReadMessage() (m *RtmpMessage, err error) {
	for m == nil {
		// chunk stream basic header.
		var fmt uint8
		var cid uint32
		if fmt, cid, err = RtmpReadBasicHeader(v.in, &v.inb); err != nil {
			if !v.closing {
				core.Warn.Println("read basic header failed. err is", err)
			}
			return
		}

		var chunk *RtmpChunk
		if c, ok := v.chunks[cid]; !ok {
			chunk = NewRtmpChunk(cid)
			v.chunks[cid] = chunk
		} else {
			chunk = c
		}

		// chunk stream message header
		if err = RtmpReadMessageHeader(v.in, &v.inb, fmt, chunk); err != nil {
			return
		}

		// read msg payload from chunk stream.
		if m, err = RtmpReadMessagePayload(v.inChunkSize, v.in, &v.inb, chunk); err != nil {
			return
		}

		// truncate the buffer.
		v.inb.Truncate(v.inb.Len())
	}

	if err = v.onRecvMessage(m); err != nil {
		return nil, err
	}

	return
}

func (v *RtmpStack) onRecvMessage(m *RtmpMessage) (err error) {
	// acknowledgement
	// TODO: FIXME: implements it.

	switch m.MessageType {
	case RtmpMsgSetChunkSize, RtmpMsgUserControlMessage, RtmpMsgWindowAcknowledgementSize:
		// we will handle these packet.
	default:
		return
	}

	var p RtmpPacket
	if p, err = v.DecodeMessage(m); err != nil {
		return
	}

	switch p := p.(type) {
	case *RtmpSetWindowAckSizePacket:
	// TODO: FIXME: implements it.
	case *RtmpSetChunkSizePacket:
		// for some server, the actual chunk size can greater than the max value(65536),
		// so we just warning the invalid chunk size, and actually use it is ok,
		// @see: https://github.com/ossrs/srs/issues/160
		if p.ChunkSize < RtmpMinChunkSize || p.ChunkSize > RtmpMaxChunkSize {
			core.Warn.Println("accept invalid chunk size", p.ChunkSize)
		}
		v.inChunkSize = uint32(p.ChunkSize)
		core.Trace.Println("input chunk size to", v.inChunkSize)
	case *RtmpUserControlPacket:
		// TODO: FIXME: implements it.
	}

	return
}

func (v *RtmpStack) SendMessage(m *RtmpMessage) (err error) {
	// we directly send out the packet,
	// use very simple algorithm, not very fast,
	// but it's ok.
	b := bytes.NewBuffer(m.Payload.Bytes())

	for b.Len() > 0 {
		// first chunk, c0.
		var vb bytes.Buffer

		// for chunk header without extended timestamp.
		if firstChunk := bool(b.Len() == m.Payload.Len()); firstChunk {
			// write new chunk stream header, fmt is 0
			if err = vb.WriteByte(byte(0x00 | (byte(m.PreferCid) & 0x3f))); err != nil {
				return
			}

			// chunk message header, 11 bytes
			// timestamp, 3bytes, big-endian
			ts := []byte{0xff, 0xff, 0xff}
			if m.Timestamp < RtmpExtendedTimestamp {
				ts[0] = byte(m.Timestamp >> 16)
				ts[1] = byte(m.Timestamp >> 8)
				ts[2] = byte(m.Timestamp)
			}
			if _, err = vb.Write(ts); err != nil {
				return
			}

			// message_length, 3bytes, big-endian
			ts[0] = byte(m.Payload.Len() >> 16)
			ts[1] = byte(m.Payload.Len() >> 8)
			ts[2] = byte(m.Payload.Len())
			if _, err = vb.Write(ts); err != nil {
				return
			}

			// message_type, 1bytes
			if err = vb.WriteByte(byte(m.MessageType)); err != nil {
				return
			}

			// stream_id, 4bytes, little-endian
			if err = binary.Write(&vb, binary.LittleEndian, m.StreamId); err != nil {
				return
			}
		} else {
			// write no message header chunk stream, fmt is 3
			// @remark, if perfer_cid > 0x3F, that is, use 2B/3B chunk header,
			// SRS will rollback to 1B chunk header.
			if err = vb.WriteByte(byte(0xC0 | (byte(m.PreferCid) & 0x3f))); err != nil {
				return
			}
		}

		// for chunk extended timestamp.
		//
		// for c0
		// chunk extended timestamp header, 0 or 4 bytes, big-endian
		//
		// for c3:
		// chunk extended timestamp header, 0 or 4 bytes, big-endian
		// 6.1.3. Extended Timestamp
		// This field is transmitted only when the normal time stamp in the
		// chunk message header is set to 0x00ffffff. If normal time stamp is
		// set to any value less than 0x00ffffff, this field MUST NOT be
		// present. This field MUST NOT be present if the timestamp field is not
		// present. Type 3 chunks MUST NOT have this field.
		// adobe changed for Type3 chunk:
		//        FMLE always sendout the extended-timestamp,
		//        must send the extended-timestamp to FMS,
		//        must send the extended-timestamp to flash-player.
		// @see: ngx_rtmp_prepare_message
		// @see: http://blog.csdn.net/win_lin/article/details/13363699
		if m.Timestamp >= RtmpExtendedTimestamp {
			if err = binary.Write(&vb, binary.LittleEndian, uint32(m.Timestamp)); err != nil {
				return
			}
		}

		// write chunk header.
		nvb := int64(vb.Len())
		if _, err = io.CopyN(v.out, &vb, nvb); err != nil {
			return
		}

		// write chunk payload
		size := int64(math.Min(float64(v.outChunkSize), float64(b.Len())))
		if _, err = io.CopyN(v.out, b, size); err != nil {
			return
		}
	}

	return
}

// read the RTMP message from buffer inb which load from reader in.
// return the completed message from chunk partial message.
func RtmpReadMessagePayload(chunkSize uint32, in io.Reader, inb *bytes.Buffer, chunk *RtmpChunk) (m *RtmpMessage, err error) {
	m = chunk.partialMessage
	if m == nil {
		panic("chunk message should never be nil")
	}

	// mix reader to read from preload body or reader.
	r := NewMixReader(inb, in)

	// the preload body must be consumed in a time.
	left := (int)(chunk.payloadLength - uint32(m.Payload.Len()))
	if chunk.payloadLength == 0 {
		// empty message
		chunk.partialMessage = nil
		return nil, nil
	}

	// the chunk payload to read this time.
	if int(chunkSize) < left {
		left = int(chunkSize)
	}

	// read payload to buffer
	if _, err = io.CopyN(&m.Payload, r, int64(left)); err != nil {
		core.Error.Println("read body failed. err is", err)
		return
	}

	// got entire RTMP message?
	if chunk.payloadLength == uint32(m.Payload.Len()) {
		chunk.partialMessage = nil
		return
	}

	// partial message.
	return nil, nil
}

// parse the message header.
//   3bytes: timestamp delta,    fmt=0,1,2
//   3bytes: payload length,     fmt=0,1
//   1bytes: message type,       fmt=0,1
//   4bytes: stream id,          fmt=0
// where:
//   fmt=0, 0x0X
//   fmt=1, 0x4X
//   fmt=2, 0x8X
//   fmt=3, 0xCX
// @remark we return the b which indicates the body read in this process,
// 		for the c3 header, we try to read more bytes which maybe header
// 		or the body.
func RtmpReadMessageHeader(in io.Reader, inb *bytes.Buffer, fmt uint8, chunk *RtmpChunk) (err error) {
	// we should not assert anything about fmt, for the first packet.
	// (when first packet, the chunk->msg is NULL).
	// the fmt maybe 0/1/2/3, the FMLE will send a 0xC4 for some audio packet.
	// the previous packet is:
	//     04                // fmt=0, cid=4
	//     00 00 1a          // timestamp=26
	//     00 00 9d          // payload_length=157
	//     08                // message_type=8(audio)
	//     01 00 00 00       // stream_id=1
	// the current packet maybe:
	//     c4             // fmt=3, cid=4
	// it's ok, for the packet is audio, and timestamp delta is 26.
	// the current packet must be parsed as:
	//     fmt=0, cid=4
	//     timestamp=26+26=52
	//     payload_length=157
	//     message_type=8(audio)
	//     stream_id=1
	// so we must update the timestamp even fmt=3 for first packet.
	//
	// fresh packet used to update the timestamp even fmt=3 for first packet.
	// fresh packet always means the chunk is the first one of message.
	isFirstMsgOfChunk := bool(chunk.partialMessage == nil)

	// but, we can ensure that when a chunk stream is fresh,
	// the fmt must be 0, a new stream.
	if chunk.isFresh && fmt != RtmpFmtType0 {
		// for librtmp, if ping, it will send a fresh stream with fmt=1,
		// 0x42             where: fmt=1, cid=2, protocol contorl user-control message
		// 0x00 0x00 0x00   where: timestamp=0
		// 0x00 0x00 0x06   where: payload_length=6
		// 0x04             where: message_type=4(protocol control user-control message)
		// 0x00 0x06            where: event Ping(0x06)
		// 0x00 0x00 0x0d 0x0f  where: event data 4bytes ping timestamp.
		// @see: https://github.com/ossrs/srs/issues/98
		if chunk.cid == RtmpCidProtocolControl && fmt == RtmpFmtType1 {
			core.Warn.Println("accept cid=2,fmt=1 to make librtmp happy.")
		} else {
			// must be a RTMP protocol level error.
			core.Error.Println("fresh chunk fmt must be", RtmpFmtType0, "actual is", fmt)
			return RtmpChunkError
		}
	}

	// when exists cache msg, means got an partial message,
	// the fmt must not be type0 which means new message.
	if !isFirstMsgOfChunk && fmt == RtmpFmtType0 {
		core.Error.Println("chunk partial msg, fmt must be", RtmpFmtType0, "actual is", fmt)
		return RtmpChunkError
	}

	// create msg when new chunk stream start
	if chunk.partialMessage == nil {
		chunk.partialMessage = NewRtmpMessage()
	}

	// read message header from socket to buffer.
	nbhs := [4]int{11, 7, 3, 0}
	nbh := nbhs[fmt]

	var bh []byte
	if nbh > 0 {
		if err = core.Grow(in, inb, nbh); err != nil {
			return
		}
		bh = inb.Next(nbh)
	}

	// parse the message header.
	//   3bytes: timestamp delta,    fmt=0,1,2
	//   3bytes: payload length,     fmt=0,1
	//   1bytes: message type,       fmt=0,1
	//   4bytes: stream id,          fmt=0
	// where:
	//   fmt=0, 0x0X
	//   fmt=1, 0x4X
	//   fmt=2, 0x8X
	//   fmt=3, 0xCX
	// see also: ngx_rtmp_recv
	if fmt <= RtmpFmtType2 {
		delta := uint32(bh[2]) | uint32(bh[1])<<8 | uint32(bh[0])<<16

		// for a message, if msg exists in cache, the delta must not changed.
		if !isFirstMsgOfChunk && chunk.timestampDelta != delta {
			core.Error.Println("chunk msg exists, should not change the delta.")
			return RtmpChunkError
		}

		// fmt: 0
		// timestamp: 3 bytes
		// If the timestamp is greater than or equal to 16777215
		// (hexadecimal 0x00ffffff), this value MUST be 16777215, and the
		// 'extended timestamp header' MUST be present. Otherwise, this value
		// SHOULD be the entire timestamp.
		//
		// fmt: 1 or 2
		// timestamp delta: 3 bytes
		// If the delta is greater than or equal to 16777215 (hexadecimal
		// 0x00ffffff), this value MUST be 16777215, and the 'extended
		// timestamp header' MUST be present. Otherwise, this value SHOULD be
		// the entire delta.
		if chunk.hasExtendedTimestamp = bool(delta >= RtmpExtendedTimestamp); !chunk.hasExtendedTimestamp {
			// no extended-timestamp, apply the delta.
			chunk.timestampDelta = delta

			// Extended timestamp: 0 or 4 bytes
			// This field MUST be sent when the normal timsestamp is set to
			// 0xffffff, it MUST NOT be sent if the normal timestamp is set to
			// anything else. So for values less than 0xffffff the normal
			// timestamp field SHOULD be used in which case the extended timestamp
			// MUST NOT be present. For values greater than or equal to 0xffffff
			// the normal timestamp field MUST NOT be used and MUST be set to
			// 0xffffff and the extended timestamp MUST be sent.
			if fmt == RtmpFmtType0 {
				// 6.1.2.1. Type 0
				// For a type-0 chunk, the absolute timestamp of the message is sent
				// here.
				chunk.timestamp = uint64(delta)
			} else {
				// 6.1.2.2. Type 1
				// 6.1.2.3. Type 2
				// For a type-1 or type-2 chunk, the difference between the previous
				// chunk's timestamp and the current chunk's timestamp is sent here.
				// @remark for continuous chunk, timestamp never change.
				if isFirstMsgOfChunk {
					chunk.timestamp += uint64(delta)
				}
			}
		}

		if fmt <= RtmpFmtType1 {
			payloadLength := uint32(bh[5]) | uint32(bh[4])<<8 | uint32(bh[3])<<16
			mtype := uint8(bh[6])

			// for a message, if msg exists in cache, the size must not changed.
			if !isFirstMsgOfChunk && chunk.payloadLength != payloadLength {
				core.Error.Println("chunk msg exists, payload length should not be changed.")
				return RtmpChunkError
			}
			// for a message, if msg exists in cache, the type must not changed.
			if !isFirstMsgOfChunk && chunk.messageType != mtype {
				core.Error.Println("chunk msg exists, type should not be changed.")
				return RtmpChunkError
			}
			chunk.payloadLength = payloadLength
			chunk.messageType = mtype

			if fmt == RtmpFmtType0 {
				// little-endian
				chunk.streamId = uint32(bh[7]) | uint32(bh[8])<<8 | uint32(bh[9])<<16 | uint32(bh[10])<<24
			}
		}
	} else {
		// update the timestamp even fmt=3 for first chunk packet
		if isFirstMsgOfChunk && !chunk.hasExtendedTimestamp {
			chunk.timestamp += uint64(chunk.timestampDelta)
		}
	}

	// read extended-timestamp
	if chunk.hasExtendedTimestamp {
		// try to read 4 bytes from stream,
		// which maybe the body or the extended-timestamp.
		if err = core.Grow(in, inb, 4); err != nil {
			return
		}
		b := inb.Bytes()

		// parse the extended-timestamp
		timestamp := uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
		// always use 31bits timestamp, for some server may use 32bits extended timestamp.
		// @see https://github.com/ossrs/srs/issues/111
		timestamp &= 0x7fffffff

		// RTMP specification and ffmpeg/librtmp is false,
		// but, adobe changed the specification, so flash/FMLE/FMS always true.
		// default to true to support flash/FMLE/FMS.
		//
		// ffmpeg/librtmp may donot send this filed, need to detect the value.
		// @see also: http://blog.csdn.net/win_lin/article/details/13363699
		// compare to the chunk timestamp, which is set by chunk message header
		// type 0,1 or 2.
		//
		// @remark, nginx send the extended-timestamp in sequence-header,
		// and timestamp delta in continue C1 chunks, and so compatible with ffmpeg,
		// that is, there is no continue chunks and extended-timestamp in nginx-rtmp.
		//
		// @remark, srs always send the extended-timestamp, to keep simple,
		// and compatible with adobe products.
		ctimestamp := uint32(chunk.timestamp) & 0x7fffffff

		// if ctimestamp<=0, the chunk previous packet has no extended-timestamp,
		// always use the extended timestamp.
		// @remark for the first chunk of message, always use the extended timestamp.
		if isFirstMsgOfChunk || ctimestamp <= 0 || ctimestamp == timestamp {
			chunk.timestamp = uint64(timestamp)
			inb.Next(4) // consume from buffer.
		}
	}

	// the extended-timestamp must be unsigned-int,
	//         24bits timestamp: 0xffffff = 16777215ms = 16777.215s = 4.66h
	//         32bits timestamp: 0xffffffff = 4294967295ms = 4294967.295s = 1193.046h = 49.71d
	// because the rtmp protocol says the 32bits timestamp is about "50 days":
	//         3. Byte Order, Alignment, and Time Format
	//                Because timestamps are generally only 32 bits long, they will roll
	//                over after fewer than 50 days.
	//
	// but, its sample says the timestamp is 31bits:
	//         An application could assume, for example, that all
	//        adjacent timestamps are within 2^31 milliseconds of each other, so
	//        10000 comes after 4000000000, while 3000000000 comes before
	//        4000000000.
	// and flv specification says timestamp is 31bits:
	//        Extension of the Timestamp field to form a SI32 value. This
	//        field represents the upper 8 bits, while the previous
	//        Timestamp field represents the lower 24 bits of the time in
	//        milliseconds.
	// in a word, 31bits timestamp is ok.
	// convert extended timestamp to 31bits.
	chunk.timestamp &= 0x7fffffff

	// copy header to msg
	chunk.partialMessage.MessageType = RtmpMessageType(chunk.messageType)
	chunk.partialMessage.Timestamp = chunk.timestamp
	chunk.partialMessage.PreferCid = chunk.cid
	chunk.partialMessage.StreamId = chunk.streamId

	// update chunk information.
	chunk.fmt = fmt
	chunk.isFresh = false
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
// @remark use inb to parse and read from in to inb if no data.
func RtmpReadBasicHeader(in io.Reader, inb *bytes.Buffer) (fmt uint8, cid uint32, err error) {
	if err = core.Grow(in, inb, 1); err != nil {
		return
	}

	var vb byte
	if vb, err = inb.ReadByte(); err != nil {
		return
	}

	fmt = uint8(vb)
	cid = uint32(fmt & 0x3f)
	fmt = (fmt >> 6) & 0x03

	// 2-63, 1B chunk header
	if cid >= 2 {
		return
	}

	// 2 or 3B
	if cid >= 0 {
		// 64-319, 2B chunk header
		if err = core.Grow(in, inb, 1); err != nil {
			return
		}

		if vb, err = inb.ReadByte(); err != nil {
			return
		}

		temp := uint32(vb) + 64

		// 64-65599, 3B chunk header
		if cid >= 1 {
			if err = core.Grow(in, inb, 1); err != nil {
				return
			}

			if vb, err = inb.ReadByte(); err != nil {
				return
			}

			temp += uint32(vb) * 256
		}

		return fmt, temp, nil
	}

	return fmt, cid, RtmpChunkError
}
