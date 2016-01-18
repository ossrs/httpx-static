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
	"bufio"
	"bytes"
	"crypto"
	"crypto/hmac"
	"crypto/sha256"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ossrs/go-oryx/core"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// error when create stream.
var createStreamError error = errors.New("rtmp create stream error")

// bytes for handshake.
type hsBytes struct {
	ctx core.Context

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

func NewHsBytes(ctx core.Context) *hsBytes {
	return &hsBytes{
		ctx: ctx,

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
	ctx := v.ctx

	select {
	case v.in <- v.C0C1():
	default:
		return core.OverflowError
	}

	core.Info.Println(ctx, "cache c0c1 ok.")
	return
}

func (v *hsBytes) readC0C1(r io.Reader) (err error) {
	ctx := v.ctx

	if v.c0c1Ok {
		return
	}

	var b bytes.Buffer
	if _, err = io.CopyN(&b, r, 1537); err != nil {
		core.Error.Println(ctx, "read c0c1 failed. err is", err)
		return
	}

	copy(v.C0C1(), b.Bytes())

	v.c0c1Ok = true
	core.Info.Println(ctx, "read c0c1 ok.")
	return
}

func (v *hsBytes) outCacheS0S1S2() (err error) {
	ctx := v.ctx

	select {
	case v.out <- v.S0S1S2():
	default:
		return core.OverflowError
	}

	core.Info.Println(ctx, "cache s0s1s2 ok.")
	return
}

func (v *hsBytes) writeS0S1S2(w io.Writer) (err error) {
	ctx := v.ctx

	r := bytes.NewReader(v.S0S1S2())
	if _, err = io.CopyN(w, r, 3073); err != nil {
		return
	}

	core.Info.Println(ctx, "write s0s1s2 ok.")
	return
}

func (v *hsBytes) inCacheC2() (err error) {
	ctx := v.ctx

	select {
	case v.in <- v.C2():
	default:
		return core.OverflowError
	}

	core.Info.Println(ctx, "cache c2 ok.")
	return
}

func (v *hsBytes) readC2(r io.Reader) (err error) {
	ctx := v.ctx

	if v.c2Ok {
		return
	}

	var b bytes.Buffer
	if _, err = io.CopyN(&b, r, 1536); err != nil {
		core.Error.Println(ctx, "read c2 failed. err is", err)
		return
	}

	copy(v.C2(), b.Bytes())

	v.c2Ok = true
	core.Info.Println(ctx, "read c2 ok.")
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

// 68bytes FMS key which is used to sign the sever packet.
var RtmpGenuineFMSKey []byte = []byte{
	0x47, 0x65, 0x6e, 0x75, 0x69, 0x6e, 0x65, 0x20,
	0x41, 0x64, 0x6f, 0x62, 0x65, 0x20, 0x46, 0x6c,
	0x61, 0x73, 0x68, 0x20, 0x4d, 0x65, 0x64, 0x69,
	0x61, 0x20, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72,
	0x20, 0x30, 0x30, 0x31, // Genuine Adobe Flash Media Server 001
	0xf0, 0xee, 0xc2, 0x4a, 0x80, 0x68, 0xbe, 0xe8,
	0x2e, 0x00, 0xd0, 0xd1, 0x02, 0x9e, 0x7e, 0x57,
	0x6e, 0xec, 0x5d, 0x2d, 0x29, 0x80, 0x6f, 0xab,
	0x93, 0xb8, 0xe6, 0x36, 0xcf, 0xeb, 0x31, 0xae,
} // 68

// 62bytes FP key which is used to sign the client packet.
var RtmpGenuineFPKey []byte = []byte{
	0x47, 0x65, 0x6E, 0x75, 0x69, 0x6E, 0x65, 0x20,
	0x41, 0x64, 0x6F, 0x62, 0x65, 0x20, 0x46, 0x6C,
	0x61, 0x73, 0x68, 0x20, 0x50, 0x6C, 0x61, 0x79,
	0x65, 0x72, 0x20, 0x30, 0x30, 0x31, // Genuine Adobe Flash Player 001
	0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8,
	0x2E, 0x00, 0xD0, 0xD1, 0x02, 0x9E, 0x7E, 0x57,
	0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
	0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
} // 62

// sha256 digest algorithm.
// @param key the sha256 key, NULL to use EVP_Digest, for instance,
//       hashlib.sha256(data).digest().
func opensslHmacSha256(key []byte, data []byte) (digest []byte, err error) {
	if key == nil {
		return crypto.SHA256.New().Sum(data), nil
	}

	h := hmac.New(sha256.New, key)
	if _, err = h.Write(data); err != nil {
		return
	}
	return h.Sum(nil), nil
}

// the schema type for complex handshake.
type chsSchema uint8

func (v chsSchema) Schema0() bool {
	return v == Schema0
}

func (v chsSchema) Schema1() bool {
	return v == Schema1
}

const (
	// c1s1 schema0
	//     key: 764bytes
	//     digest: 764bytes
	Schema0 chsSchema = iota
	// c1s1 schema1
	//     digest: 764bytes
	//     key: 764bytes
	// @remark FMS only support schema1, please read
	// 		http://blog.csdn.net/win_lin/article/details/13006803
	Schema1
)

// 764bytes key structure
//     random-data: (offset)bytes
//     key-data: 128bytes
//     random-data: (764-offset-128-4)bytes
//     offset: 4bytes
// @see also: http://blog.csdn.net/win_lin/article/details/13006803
type chsKey []byte

func (v chsKey) Random0() []byte {
	return v[0:v.Offset()]
}

func (v chsKey) Key() []byte {
	return v[v.Offset() : v.Offset()+128]
}

func (v chsKey) Random1() []byte {
	return v[v.Offset()+128 : 760]
}

func (v chsKey) Offset() uint32 {
	max := uint32(764 - 128 - 4)
	b := v[764-4 : 764]

	var offset uint32
	offset += uint32(b[0])
	offset += uint32(b[1])
	offset += uint32(b[2])
	offset += uint32(b[3])

	return offset % max
}

// 764bytes digest structure
//     offset: 4bytes
//     random-data: (offset)bytes
//     digest-data: 32bytes
//     random-data: (764-4-offset-32)bytes
// @see also: http://blog.csdn.net/win_lin/article/details/13006803
type chsDigest []byte

func (v chsDigest) Random0() []byte {
	return v[4 : 4+v.Offset()]
}

func (v chsDigest) Digest() []byte {
	return v[4+v.Offset() : 4+v.Offset()+32]
}

func (v chsDigest) Random1() []byte {
	return v[4+v.Offset()+32 : 764]
}

func (v chsDigest) Offset() uint32 {
	max := uint32(764 - 32 - 4)
	b := v[0:4]

	var offset uint32
	offset += uint32(b[0])
	offset += uint32(b[1])
	offset += uint32(b[2])
	offset += uint32(b[3])

	return offset % max
}

// c1s1 schema0
//     time: 4bytes
//     version: 4bytes
//     key: 764bytes
//     digest: 764bytes
// c1s1 schema1
//     time: 4bytes
//     version: 4bytes
//     digest: 764bytes
//     key: 764bytes
// @see also: http://blog.csdn.net/win_lin/article/details/13006803
type chsC1S1 struct {
	time    uint32
	version uint32

	// schema 0 or schema 1
	schema chsSchema
	c1s1   []byte

	key    chsKey
	digest chsDigest
}

func (v *chsC1S1) Parse(c1s1 []byte, schema chsSchema) (err error) {
	if v.c1s1 = c1s1; len(c1s1) != 1536 {
		return fmt.Errorf("c1/s1 must be 1536 bytes, actual is %v bytes", len(c1s1))
	}

	b := bytes.NewBuffer(c1s1)
	if err = binary.Read(b, binary.BigEndian, &v.time); err != nil {
		return
	}
	if err = binary.Read(b, binary.BigEndian, &v.version); err != nil {
		return
	}

	p := b.Bytes()
	if v.schema = schema; v.schema.Schema0() {
		v.key = chsKey(p[:764])
		v.digest = chsDigest(p[764:])
	} else {
		v.digest = chsDigest(p[:764])
		v.key = chsKey(p[764:])
	}
	return
}

func (v *chsC1S1) S1Create(s1 []byte, time, version uint32, c1 *chsC1S1) (err error) {
	v.schema = c1.schema
	v.c1s1 = s1[:]

	v.time = uint32(time)
	v.version = version

	var b bytes.Buffer
	if err = binary.Write(&b, binary.BigEndian, v.time); err != nil {
		return
	}
	if err = binary.Write(&b, binary.BigEndian, v.version); err != nil {
		return
	}
	copy(v.c1s1[0:8], b.Bytes())

	p := v.c1s1[8:]
	if v.schema.Schema0() {
		v.key = chsKey(p[:764])
		v.digest = chsDigest(p[764:])
	} else {
		v.digest = chsDigest(p[:764])
		v.key = chsKey(p[764:])
	}

	// use openssl DH to
	// 		1. generate public and private key, save to s1 object.
	// 		2. compute the shared key, copy to s1.key.
	//		3. client use shared key to communicate.
	// where the shared key is computed by client and server public key.
	// for currently we don't use the shared key,
	// so we just use any random number.
	// TODO: generate and compute the real shared key.

	// digest s1.
	var checksum []byte
	if checksum, err = v.digestS1(); err != nil {
		return
	}

	// copy digest to s1.
	_ = copy(v.digest.Digest(), checksum[0:32])
	return
}

func (v *chsC1S1) Validate() (ok bool, err error) {
	var checksum []byte
	if checksum, err = v.digestC1(); err != nil {
		return
	}

	expect := v.digest.Digest()
	ok = bytes.Equal(checksum[0:32], expect)
	return
}

func (v *chsC1S1) digestOffset() int {
	if v.schema.Schema0() {
		return 8 + 764 + 4 + int(v.digest.Offset())
	}
	return 8 + 4 + int(v.digest.Offset())
}

func (v *chsC1S1) part1() []byte {
	if v.schema.Schema0() {
		return v.c1s1[0:v.digestOffset()]
	}
	return v.c1s1[0:v.digestOffset()]
}

func (v *chsC1S1) part2() []byte {
	if v.schema.Schema0() {
		return v.c1s1[v.digestOffset()+32:]
	}
	return v.c1s1[v.digestOffset()+32:]
}

func (v *chsC1S1) digestC1() ([]byte, error) {
	// c1s1 is splited by digest:
	//     c1s1-part1: n bytes (time, version, key and digest-part1).
	//     digest-data: 32bytes
	//     c1s1-part2: (1536-n-32)bytes (digest-part2)
	join := append([]byte{}, v.part1()...)
	join = append(join, v.part2()...)
	return opensslHmacSha256(RtmpGenuineFPKey[0:30], join)
}

func (v *chsC1S1) digestS1() ([]byte, error) {
	// c1s1 is splited by digest:
	//     c1s1-part1: n bytes (time, version, key and digest-part1).
	//     digest-data: 32bytes
	//     c1s1-part2: (1536-n-32)bytes (digest-part2)
	join := append([]byte{}, v.part1()...)
	join = append(join, v.part2()...)
	return opensslHmacSha256(RtmpGenuineFMSKey[0:36], join)
}

// the c2s2 complex handshake structure.
// random-data: 1504bytes
// digest-data: 32bytes
// @see also: http://blog.csdn.net/win_lin/article/details/13006803
type chsC2S2 struct {
	c2s2 []byte
}

func (v *chsC2S2) Random() []byte {
	return v.c2s2[0:1504]
}

func (v *chsC2S2) Digest() []byte {
	return v.c2s2[1504:1536]
}

func (v *chsC2S2) Parse(c2s2 []byte) (err error) {
	v.c2s2 = c2s2
	return
}

func (v *chsC2S2) S2Create(s2 []byte, c1 *chsC1S1) (err error) {
	v.c2s2 = s2

	var tempKey []byte
	if tempKey, err = opensslHmacSha256(RtmpGenuineFMSKey[0:68], c1.digest.Digest()); err != nil {
		return
	}

	var digest []byte
	if digest, err = opensslHmacSha256(tempKey[0:32], v.Random()); err != nil {
		return
	}

	_ = copy(v.Digest(), digest[0:32])

	return
}

// rtmp request.
type RtmpRequest struct {
	ctx core.Context

	// the tcUrl in RTMP connect app request.
	TcUrl string
	// the required object encoding.
	ObjectEncoding float64

	// the stream to publish or play.
	Stream string
	// the type of connection, publish or play.
	Type RtmpConnType
	// for play, the duration.
	Duration float64

	// the vhost parsed from tcUrl or stream.
	Vhost string
	// the app parsed from tcUrl.
	App string
	// the url, parsed for tcUrl/stream?params.
	Url *url.URL
}

func NewRtmpRequest(ctx core.Context) *RtmpRequest {
	return &RtmpRequest{
		ctx:  ctx,
		Type: RtmpUnknown,
		Url:  &url.URL{},
	}
}

// the uri to identify the request, vhost/app/stream.
func (v *RtmpRequest) Uri() string {
	uri := ""
	if v.Vhost != core.RtmpDefaultVhost {
		uri += v.Vhost
	}

	uri += "/" + v.App
	uri += "/" + v.Stream

	return uri
}

// the rtmp port, default to 1935.
func (v *RtmpRequest) Port() int {
	if _, p, err := net.SplitHostPort(v.Url.Host); err != nil {
		return core.RtmpListen
	} else if p, err := strconv.ParseInt(p, 10, 32); err != nil {
		return core.RtmpListen
	} else if p <= 0 {
		return core.RtmpListen
	} else {
		return int(p)
	}
}

// the host connected at, the ip or domain name(vhost).
func (v *RtmpRequest) Host() string {
	if !strings.Contains(v.Url.Host, ":") {
		return v.Url.Host
	}

	if h, _, err := net.SplitHostPort(v.Url.Host); err != nil {
		return ""
	} else {
		return h
	}
}

// parse the rtmp request object from tcUrl/stream?params
// to finger it out the vhost and url.
func (v *RtmpRequest) Reparse() (err error) {
	ctx := v.ctx

	// convert app...pn0...pv0...pn1...pv1...pnn...pvn
	// to (without space):
	// 		app ? pn0=pv0 && pn1=pv1 && pnn=pvn
	// where ... can replaced by ___ or ? or && or &
	mfn := func(s string) string {
		r := s
		matchs := []string{"...", "___", "?", "&&", "&"}
		for _, m := range matchs {
			r = strings.Replace(r, m, "...", -1)
		}
		return r
	}
	ffn := func(s string) string {
		r := mfn(s)
		for first := true; ; first = false {
			if !strings.Contains(r, "...") {
				break
			}
			if first {
				r = strings.Replace(r, "...", "?", 1)
			} else {
				r = strings.Replace(r, "...", "&&", 1)
			}

			if !strings.Contains(r, "...") {
				break
			}
			r = strings.Replace(r, "...", "=", 1)
		}
		return r
	}

	// format the app and stream.
	v.TcUrl = ffn(v.TcUrl)
	v.Stream = ffn(v.Stream)

	// format the tcUrl and stream.
	var params string
	if ss := strings.SplitN(v.TcUrl, "?", 2); len(ss) == 2 {
		v.TcUrl = ss[0]
		params = ss[1]
	}
	if ss := strings.SplitN(v.Stream, "?", 2); len(ss) == 2 {
		v.Stream = ss[0]
		params += "&&" + ss[1]
	}
	params = strings.TrimLeft(params, "&&")

	// the standard rtmp uri is:
	//		rtmp://ip:port/app?params
	// where the simple url is:
	//		rtmp://vhost/app/stream
	// and the standard adobe url to support param is:
	//		rtmp://ip/app?params/stream
	// some client use stream to pass the params:
	//		rtmp://ip/app/stream?params
	// we will parse all uri to the standard rtmp uri.
	u := fmt.Sprintf("%v?%v", v.TcUrl, params)
	if v.Url, err = url.Parse(u); err != nil {
		return
	}
	q := v.Url.Query()

	// parse result.
	v.Vhost = v.Host()
	if p := q.Get("vhost"); p != "" {
		v.Vhost = p
	} else if p := q.Get("domain"); p != "" {
		v.Vhost = p
	}

	if v.App = strings.TrimLeft(v.Url.Path, "/"); v.App == "" {
		v.App = core.RtmpDefaultApp
	}
	v.Stream = strings.Trim(v.Stream, "/")

	// check.
	if v.Vhost == "" {
		core.Error.Println(ctx, "vhost must not be empty")
		return RequestUrlError
	}
	if v.App == "" && v.Stream == "" {
		core.Error.Println(ctx, "both app and stream must not be empty")
		return RequestUrlError
	}
	if p := v.Port(); p <= 0 {
		core.Error.Println(ctx, "port must be positive, actual is", p)
		return RequestUrlError
	}

	return
}

// the rtmp client type.
type RtmpConnType uint8

func (v RtmpConnType) String() string {
	switch v {
	case RtmpPlay:
		return "play"
	case RtmpFmlePublish:
		return "fmle-publish"
	case RtmpFlashPublish:
		return "flash-publish"
	default:
		return "unknown"
	}
}

// whether connection is player
func (v RtmpConnType) IsPlay() bool {
	return v == RtmpPlay
}

// whether connection is flash or fmle publisher.
func (v RtmpConnType) IsPublish() bool {
	return v == RtmpFlashPublish || v == RtmpFmlePublish
}

const (
	RtmpUnknown RtmpConnType = iota
	RtmpPlay
	RtmpFmlePublish
	RtmpFlashPublish
)

// rtmp protocol stack.
type RtmpConnection struct {
	ctx core.Context

	// the current using stream id.
	sid uint32
	// the rtmp request.
	Req *RtmpRequest

	// to receive the quit message from server.
	wc core.WorkerContainer
	// the handshake bytes for RTMP.
	handshake *hsBytes
	// the underlayer transport.
	transport io.ReadWriteCloser
	// when handshake and negotiate,
	// the connection must send message one when got it,
	// while we can group messages when send audio/video stream.
	groupMessages   bool
	nbGroupMessages int
	// the RTMP protocol stack.
	stack *RtmpStack
	// input channel, receive message from client.
	in chan *RtmpMessage
	// whether receiver and sender already quit.
	workers sync.WaitGroup

	// whether closed.
	closed bool
	// the locker for close
	closeLock sync.Mutex
	// when receiver or sender quit, notify main goroutine.
	closing *core.Quiter

	// the locker for write.
	writeLock sync.Mutex
	// use cache queue.
	out []*RtmpMessage

	// whether stack should flush all messages.
	shouldFlush chan bool
	// whether the sender need to be notified.
	needNotifyFlusher bool
	// whether the sender is working.
	isFlusherWorking bool
}

func NewRtmpConnection(ctx core.Context, transport io.ReadWriteCloser, wc core.WorkerContainer) *RtmpConnection {
	v := &RtmpConnection{
		ctx:           ctx,
		sid:           0,
		Req:           NewRtmpRequest(ctx),
		wc:            wc,
		handshake:     NewHsBytes(ctx),
		transport:     transport,
		groupMessages: false,
		stack:         NewRtmpStack(ctx, transport, transport),
		in:            make(chan *RtmpMessage, RtmpInCache),
		out:           make([]*RtmpMessage, 0, RtmpDefaultMwMessages*2),
		closing:       core.NewQuiter(),
		shouldFlush:   make(chan bool, 1),
	}

	// wait for goroutine to run.
	wait := make(chan bool)

	// start the receiver and sender.
	// directly use raw goroutine, for donot cause the container to quit.
	go core.Recover(ctx, "rtmp receiver", func() error {
		v.workers.Add(1)
		defer v.workers.Done()

		// noitfy the main goroutine to quit.
		defer func() {
			v.closing.Quit()
		}()

		// notify the main goroutine the receiver is ok.
		wait <- true

		if err := v.receiver(); err != nil {
			return err
		}
		return nil
	})
	go core.Recover(ctx, "rtmp sender", func() error {
		v.workers.Add(1)
		defer v.workers.Done()

		// noitfy the main goroutine to quit.
		defer func() {
			v.closing.Quit()
		}()

		// notify the main goroutine the sender is ok.
		wait <- true

		// when got quit message, close the underlayer transport.
		select {
		case <-v.wc.QC():
			v.wc.Quit()
			return v.transport.Close()
		case <-v.closing.QC():
			return v.closing.Quit()
		}
	})

	// wait for receiver and sender ok.
	<-wait
	<-wait

	// handle reload.
	core.Conf.Subscribe(v)

	return v
}

// retrieve the context of connection.
func (v *RtmpConnection) Ctx() core.Context {
	return v.ctx
}

// close the connection to client.
// TODO: FIXME: should be thread safe.
func (v *RtmpConnection) Close() {
	ctx := v.ctx

	v.closeLock.Lock()
	defer v.closeLock.Unlock()

	if v.closed {
		return
	}
	v.closed = true

	// unhandle reload.
	core.Conf.Unsubscribe(v)

	// close the underlayer transport.
	_ = v.transport.Close()
	core.Info.Println(ctx, "close the transport")

	// notify other goroutine to close.
	v.closing.Quit()
	core.Info.Println(ctx, "set closed and wait.")

	// wait for sender and receiver to quit.
	v.workers.Wait()
	core.Trace.Println(ctx, "closed")

	return
}

// interface ReloadHandler
func (v *RtmpConnection) OnReloadGlobal(scope int, cc, pc *core.Config) (err error) {
	return
}

func (v *RtmpConnection) OnReloadVhost(vhost string, scope int, cc, pc *core.Config) (err error) {
	if vhost == v.Req.Vhost && scope == core.ReloadMwLatency {
		return v.updateNbGroupMessages()
	}
	return
}

// handshake with client, try complex then simple.
func (v *RtmpConnection) Handshake() (err error) {
	ctx := v.ctx

	// got c0c1.
	if err = v.waitC0C1(); err != nil {
		return
	}

	// create s0s1s2 from c1.
	v.handshake.createS0S1S2()

	// complex handshake.
	chs := func() (completed bool, err error) {
		c1 := &chsC1S1{}

		// try schema0.
		// @remark, use schema0 to make flash player happy.
		if err = c1.Parse(v.handshake.C1(), Schema0); err != nil {
			return
		}
		if completed, err = c1.Validate(); err != nil {
			return
		}

		// try schema1
		if !completed {
			if err = c1.Parse(v.handshake.C1(), Schema1); err != nil {
				return
			}
			if completed, err = c1.Validate(); err != nil {
				return
			}
		}

		// encode s1
		s1 := &chsC1S1{}
		time := uint32(time.Now().Unix())
		version := uint32(0x01000504) // server s1 version
		if err = s1.S1Create(v.handshake.S1(), time, version, c1); err != nil {
			return
		}

		// encode s2
		s2 := &chsC2S2{}
		if err = s2.S2Create(v.handshake.S2(), c1); err != nil {
			return
		}
		return
	}

	// simple handshake.
	shs := func() (err error) {
		// plain text required.
		if !v.handshake.ClientPlaintext() {
			return fmt.Errorf("only support rtmp plain text.")
		}

		return
	}

	// try complex, then simple handshake.
	var completed bool
	if completed, err = chs(); err != nil {
		return
	}
	if !completed {
		core.Trace.Println(ctx, "rollback to simple handshake.")
		if err = shs(); err != nil {
			return
		}
	} else {
		core.Trace.Println(ctx, "complex handshake ok.")
	}

	// cache the s0s1s2 for sender to write.
	if err = v.handshake.outCacheS0S1S2(); err != nil {
		return
	}
	// we must manually send out the s0s1s2,
	// for the writer is belong to main goroutine.
	if err = v.handshake.writeS0S1S2(v.transport); err != nil {
		return
	}

	// got c2.
	if err = v.waitC2(); err != nil {
		return
	}

	return
}

func (v *RtmpConnection) waitC0C1() (err error) {
	ctx := v.ctx

	// use short handshake timeout.
	timeout := HandshakeTimeout

	// wait c0c1
	select {
	case <-v.handshake.in:
		break
	case <-time.After(timeout):
		core.Error.Println(ctx, "c0c1 timeout for", timeout)
		return core.TimeoutError
	case <-v.closing.QC():
		return v.closing.Quit()
	case <-v.wc.QC():
		return v.wc.Quit()
	}

	return
}

func (v *RtmpConnection) waitC2() (err error) {
	ctx := v.ctx

	// use short handshake timeout.
	timeout := HandshakeTimeout

	// wait c2
	select {
	case <-v.handshake.in:
		break
	case <-time.After(timeout):
		core.Error.Println(ctx, "c2 timeout for", timeout)
		return core.TimeoutError
	case <-v.closing.QC():
		return v.closing.Quit()
	case <-v.wc.QC():
		return v.wc.Quit()
	}

	return
}

// do connect app with client, to discovery tcUrl.
func (v *RtmpConnection) ExpectConnectApp(r *RtmpRequest) (err error) {
	ctx := v.ctx

	// connect(tcUrl)
	return v.read(ConnectAppTimeout, func(m *RtmpMessage) (loop bool, err error) {
		var p RtmpPacket
		if p, err = v.stack.DecodeMessage(m); err != nil {
			return
		}
		if p, ok := p.(*RtmpConnectAppPacket); ok {
			if p, ok := p.CommandObject.Get("tcUrl").(*Amf0String); ok {
				r.TcUrl = string(*p)
			}
			if p, ok := p.CommandObject.Get("objectEncoding").(*Amf0Number); ok {
				r.ObjectEncoding = float64(*p)
			}

			objectEncoding := fmt.Sprintf("AMF%v", int(r.ObjectEncoding))
			core.Trace.Println(ctx, "connect at", r.TcUrl, objectEncoding)
		} else {
			// try next.
			return true, nil
		}
		return
	})
}

// set ack size to client, client will send ack-size for each ack window
func (v *RtmpConnection) SetWindowAckSize(ack uint32) (err error) {
	p := NewRtmpSetWindowAckSizePacket().(*RtmpSetWindowAckSizePacket)
	p.Ack = RtmpUint32(ack)

	return v.write(p, 0)
}

// @type: The sender can mark this message hard (0), soft (1), or dynamic (2)
// using the Limit type field.
func (v *RtmpConnection) SetPeerBandwidth(bw uint32, t uint8) (err error) {
	p := NewRtmpSetPeerBandwidthPacket().(*RtmpSetPeerBandwidthPacket)
	p.Bandwidth = RtmpUint32(bw)
	p.Type = RtmpUint8(t)

	return v.write(p, 0)
}

// set the chunk size.
func (v *RtmpConnection) SetChunkSize(n int) (err error) {
	p := NewRtmpSetChunkSizePacket().(*RtmpSetChunkSizePacket)
	p.ChunkSize = RtmpUint32(n)

	return v.write(p, 0)
}

// @param server_ip the ip of server.
func (v *RtmpConnection) ResponseConnectApp() (err error) {
	p := NewRtmpConnectAppResPacket().(*RtmpConnectAppResPacket)

	p.Props.Set("fmsVer", NewAmf0String(fmt.Sprintf("FMS/%v", RtmpSigFmsVer)))
	p.Props.Set("capabilities", NewAmf0Number(127))
	p.Props.Set("mode", NewAmf0Number(1))

	p.Info.Set(StatusLevel, NewAmf0String(StatusLevelStatus))
	p.Info.Set(StatusCode, NewAmf0String(StatusCodeConnectSuccess))
	p.Info.Set(StatusDescription, NewAmf0String("Connection succeeded"))
	p.Info.Set("objectEncoding", NewAmf0Number(v.Req.ObjectEncoding))

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

	return v.write(p, 0)
}

// response client the onBWDone message.
func (v *RtmpConnection) OnBwDone() (err error) {
	p := NewRtmpOnBwDonePacket().(*RtmpOnBwDonePacket)

	return v.write(p, 0)
}

// recv some message to identify the client.
// @stream_id, client will createStream to play or publish by flash,
//         the stream_id used to response the createStream request.
// @type, output the client type.
// @stream_name, output the client publish/play stream name. @see: SrsRequest.stream
// @duration, output the play client duration. @see: SrsRequest.duration
func (v *RtmpConnection) Identify() (connType RtmpConnType, streamName string, duration float64, err error) {
	ctx := v.ctx

	err = v.identify(func(p RtmpPacket) (loop bool, err error) {
		switch p := p.(type) {
		case *RtmpCreateStreamPacket:
			core.Info.Println(ctx, "identify createStream")
			connType, streamName, duration, err = v.identifyCreateStream(nil, p)
			return
		case *RtmpFMLEStartPacket:
			core.Info.Println(ctx, "identify fmlePublish")
			connType, streamName, err = v.identifyFmlePublish(p)
			return
		case *RtmpPlayPacket:
			core.Info.Println(ctx, "identify play")
			connType, streamName, duration, err = v.identifyPlay(p)
			return
		}
		// TODO: FIXME: implements it.

		// for other call msgs,
		// support response null first,
		// @see https://github.com/ossrs/srs/issues/106
		// TODO: FIXME: response in right way, or forward in edge mode.
		if p, ok := p.(*RtmpCallPacket); ok {
			res := NewRtmpCallResPacket().(*RtmpCallResPacket)
			res.TransactionId = p.TransactionId
			if err = v.write(res, 0); err != nil {
				core.Error.Println(ctx, "response call failed. err is", err)
				return
			}
		}

		return true, nil
	})

	return
}

// when request parsed, notify the connection.
func (v *RtmpConnection) OnUrlParsed() (err error) {
	return v.updateNbGroupMessages()
}
func (v *RtmpConnection) updateNbGroupMessages() (err error) {
	if v.nbGroupMessages, err = core.Conf.VhostGroupMessages(v.Req.Vhost); err != nil {
		return
	}
	return
}

// for Flash encoder, response the start publish event.
func (v *RtmpConnection) FlashStartPublish() (err error) {
	ctx := v.ctx

	res := NewRtmpOnStatusCallPacket().(*RtmpOnStatusCallPacket)
	res.Data.Set(StatusLevel, NewAmf0String(StatusLevelStatus))
	res.Data.Set(StatusCode, NewAmf0String(StatusCodePublishStart))
	res.Data.Set(StatusDescription, NewAmf0String("Started publishing stream."))
	res.Data.Set(StatusClientId, NewAmf0String(RtmpSigClientId))
	if err = v.write(res, v.sid); err != nil {
		return
	}

	core.Trace.Println(ctx, "Flash start publish ok.")
	return
}

// for FMLE encoder, response the start publish event.
func (v *RtmpConnection) FmleStartPublish() (err error) {
	ctx := v.ctx

	return v.read(FmlePublishTimeout, func(m *RtmpMessage) (loop bool, err error) {
		var p RtmpPacket
		if p, err = v.stack.DecodeMessage(m); err != nil {
			return
		}

		switch p := p.(type) {
		case *RtmpFMLEStartPacket:
			res := NewRtmpFMLEStartResPacket().(*RtmpFMLEStartResPacket)
			res.TransactionId = p.TransactionId
			if err = v.write(res, 0); err != nil {
				return
			}
			return true, nil
		case *RtmpCreateStreamPacket:
			// increasing the stream id.
			v.sid++

			res := NewRtmpCreateStreamResPacket().(*RtmpCreateStreamResPacket)
			res.TransactionId = p.TransactionId
			res.StreamId = Amf0Number(v.sid)

			if err = v.write(res, 0); err != nil {
				return
			}
			return true, nil
		case *RtmpPublishPacket:
			res := NewRtmpOnStatusCallPacket().(*RtmpOnStatusCallPacket)
			res.Name = Amf0String(Amf0CommandFcPublish)
			res.Data.Set(StatusCode, NewAmf0String(StatusCodePublishStart))
			res.Data.Set(StatusDescription, NewAmf0String("Started publishing stream."))
			if err = v.write(res, v.sid); err != nil {
				return
			}

			res = NewRtmpOnStatusCallPacket().(*RtmpOnStatusCallPacket)
			res.Data.Set(StatusLevel, NewAmf0String(StatusLevelStatus))
			res.Data.Set(StatusCode, NewAmf0String(StatusCodePublishStart))
			res.Data.Set(StatusDescription, NewAmf0String("Started publishing stream."))
			res.Data.Set(StatusClientId, NewAmf0String(RtmpSigClientId))
			if err = v.write(res, v.sid); err != nil {
				return
			}
			return true, nil
		case *RtmpOnStatusCallPacket:
			if p.Name == "onFCPublish" {
				core.Trace.Println(ctx, "FMLE start publish ok.")
				return false, nil
			}
			core.Info.Println(ctx, "drop FMLE command", p.Name)
			return true, nil
		default:
			return true, nil
		}
	})
}

// for FMLE encoder, to unpublish.
func (v *RtmpConnection) FmleUnpublish(upp *RtmpFMLEStartPacket) (err error) {
	// publish response onFCUnpublish(NetStream.unpublish.Success)
	if res, ok := NewRtmpOnStatusCallPacket().(*RtmpOnStatusCallPacket); ok {
		res.Name = Amf0String(Amf0CommandOnFcUnpublish)
		res.Data.Set(StatusCode, NewAmf0String(StatusCodeUnpublishSuccess))
		res.Data.Set(StatusDescription, NewAmf0String("Stop publishing stream."))
		if err = v.write(res, v.sid); err != nil {
			return
		}
	}

	// FCUnpublish response
	if res, ok := NewRtmpFMLEStartResPacket().(*RtmpFMLEStartResPacket); ok {
		res.TransactionId = upp.TransactionId
		if err = v.write(res, v.sid); err != nil {
			return
		}
	}

	// publish response onStatus(NetStream.Unpublish.Success)
	if res, ok := NewRtmpOnStatusCallPacket().(*RtmpOnStatusCallPacket); ok {
		res.Name = Amf0String(Amf0CommandOnFcUnpublish)
		res.Data.Set(StatusCode, NewAmf0String(StatusCodeUnpublishSuccess))
		res.Data.Set(StatusDescription, NewAmf0String("Stream is now unpublished"))
		res.Data.Set(StatusClientId, NewAmf0String(RtmpSigClientId))
		if err = v.write(res, v.sid); err != nil {
			return
		}
	}

	return
}

// for Flash player or edge, response the start play event.
func (v *RtmpConnection) FlashStartPlay() (err error) {
	ctx := v.ctx

	// StreamBegin
	if p, ok := NewRtmpUserControlPacket().(*RtmpUserControlPacket); ok {
		p.EventType = RtmpUint16(RtmpPcucStreamBegin)
		p.EventData = RtmpUint32(v.sid)
		if err = v.write(p, 0); err != nil {
			return
		}
	}

	// onStatus(NetStream.Play.Reset)
	if p, ok := NewRtmpOnStatusCallPacket().(*RtmpOnStatusCallPacket); ok {
		p.Data.Set(StatusLevel, NewAmf0String(StatusLevelStatus))
		p.Data.Set(StatusCode, NewAmf0String(StatusCodeStreamReset))
		p.Data.Set(StatusDescription, NewAmf0String("Playing and resetting stream."))
		p.Data.Set(StatusDetails, NewAmf0String("stream"))
		p.Data.Set(StatusClientId, NewAmf0String(RtmpSigClientId))
		if err = v.write(p, v.sid); err != nil {
			return
		}
	}

	// onStatus(NetStream.Play.Start)
	if p, ok := NewRtmpOnStatusCallPacket().(*RtmpOnStatusCallPacket); ok {
		p.Data.Set(StatusLevel, NewAmf0String(StatusLevelStatus))
		p.Data.Set(StatusCode, NewAmf0String(StatusCodeStreamStart))
		p.Data.Set(StatusDescription, NewAmf0String("Started playing stream."))
		p.Data.Set(StatusDetails, NewAmf0String("stream"))
		p.Data.Set(StatusClientId, NewAmf0String(RtmpSigClientId))
		if err = v.write(p, v.sid); err != nil {
			return
		}
	}

	// |RtmpSampleAccess(false, false)
	if p, ok := NewRtmpSampleAccessPacket().(*RtmpSampleAccessPacket); ok {
		// allow audio/video sample.
		// @see: https://github.com/ossrs/srs/issues/49
		p.VideoSampleAccess = true
		p.AudioSampleAccess = true
		if err = v.write(p, v.sid); err != nil {
			return
		}
	}

	// onStatus(NetStream.Data.Start)
	if p, ok := NewRtmpOnStatusDataPacket().(*RtmpOnStatusDataPacket); ok {
		p.Data.Set(StatusCode, NewAmf0String(StatusCodeDataStart))
		if err = v.write(p, v.sid); err != nil {
			return
		}
	}

	// ok, we enter group message mode.
	var r bool
	if r, err = core.Conf.VhostRealtime(v.Req.Vhost); err != nil {
		return
	} else if !r {
		v.groupMessages = true
	} else {
		v.groupMessages = false
		core.Trace.Println(ctx, "enter realtime mode, disable message group")
	}

	return
}

// the rtmp connection never provides send message,
// but we use cache message and the main goroutine of connection
// will use Cycle to flush messages.
func (v *RtmpConnection) CacheMessage(m *RtmpMessage) (err error) {
	v.writeLock.Lock()
	defer v.writeLock.Unlock()

	// push to queue.
	v.out = append(v.out, m)

	// notify when messages is enough and sender is not working.
	if len(v.out) >= v.requiredMessages() && !v.isFlusherWorking {
		v.needNotifyFlusher = true
	}

	// unblock the sender when got enough messages.
	if v.toggleNotify() {
		select {
		case v.shouldFlush <- true:
		default:
		}
	}

	return
}

// cycle to flush messages, and callback the fn when got message from peer.
func (v *RtmpConnection) Cycle(fn func(*RtmpMessage) error) (err error) {
	for {
		select {
		case <-v.shouldFlush:
			if err = v.flush(); err != nil {
				return
			}
		case m := <-v.in:
			if err = fn(m); err != nil {
				return
			}
		case <-v.closing.QC():
			return v.closing.Quit()
		case <-v.wc.QC():
			return v.wc.Quit()
		}
	}

	return
}

func (v *RtmpConnection) requiredMessages() int {
	if v.groupMessages && v.nbGroupMessages > 0 {
		return v.nbGroupMessages
	}
	return 1
}

func (v *RtmpConnection) toggleNotify() bool {
	nn := v.needNotifyFlusher
	v.needNotifyFlusher = false
	return nn
}

// to push message to send queue.
func (v *RtmpConnection) flush() (err error) {
	v.isFlusherWorking = true
	defer func() {
		v.isFlusherWorking = false
	}()

	for {
		// cache the required messages.
		required := v.requiredMessages()

		// force to ignore small pieces for group message.
		if v.groupMessages && len(v.out) < v.nbGroupMessages/2 {
			break
		}

		// copy messages to send.
		var out []*RtmpMessage
		func() {
			v.writeLock.Lock()
			defer v.writeLock.Unlock()

			out = v.out[:]
			v.out = v.out[0:0]
		}()

		// sendout all messages.
		for {
			// nothing, ingore.
			if len(out) == 0 {
				return
			}

			// send one by one.
			if required <= 1 {
				for _, m := range out {
					if err = v.stack.SendMessage(m); err != nil {
						return
					}
				}
				break
			}

			// last group and left messages.
			if err = v.stack.SendMessage(out...); err != nil {
				return
			}
			break
		}
	}

	return
}

// to receive message from rtmp.
func (v *RtmpConnection) RecvMessage(timeout time.Duration, fn func(*RtmpMessage) error) (err error) {
	return v.read(timeout, func(m *RtmpMessage) (loop bool, err error) {
		return true, fn(m)
	})
}

// to decode the message to packet.
func (v *RtmpConnection) DecodeMessage(m *RtmpMessage) (p RtmpPacket, err error) {
	return v.stack.DecodeMessage(m)
}

func (v *RtmpConnection) identifyCreateStream(p0, p1 *RtmpCreateStreamPacket) (connType RtmpConnType, streamName string, duration float64, err error) {
	ctx := v.ctx
	current := p1

	if csr := NewRtmpCreateStreamResPacket().(*RtmpCreateStreamResPacket); csr != nil {
		// increasing the stream id.
		v.sid++

		csr.TransactionId = current.TransactionId
		csr.StreamId = Amf0Number(float64(v.sid))
		if err = v.write(csr, 0); err != nil {
			core.Error.Println(ctx, "response createStream failed. err is", err)
			return
		}
	}

	err = v.identify(func(p RtmpPacket) (loop bool, err error) {
		switch p := p.(type) {
		case *RtmpPlayPacket:
			connType, streamName, duration, err = v.identifyPlay(p)
			return
		case *RtmpPublishPacket:
			connType, streamName, err = v.identifyFlashPublish(p)
			return
		case *RtmpCreateStreamPacket:
			// to avoid stack overflow attach.
			if p0 != nil {
				err = createStreamError
				core.Error.Println(ctx, "only support two createStream packet. err is", err)
				return
			}

			connType, streamName, duration, err = v.identifyCreateStream(current, p)
			return
		}
		return
	})

	return
}

func (v *RtmpConnection) identifyFlashPublish(p *RtmpPublishPacket) (connType RtmpConnType, streamName string, err error) {
	return RtmpFlashPublish, string(p.Stream), nil
}

func (v *RtmpConnection) identifyFmlePublish(p *RtmpFMLEStartPacket) (connType RtmpConnType, streamName string, err error) {
	ctx := v.ctx

	connType = RtmpFmlePublish
	streamName = string(p.Stream)

	res := NewRtmpFMLEStartResPacket().(*RtmpFMLEStartResPacket)
	res.TransactionId = p.TransactionId

	if err = v.write(res, 0); err != nil {
		core.Error.Println(ctx, "response identify fmle failed. err is", err)
		return
	}

	return
}

func (v *RtmpConnection) identifyPlay(p *RtmpPlayPacket) (connType RtmpConnType, streamName string, duration float64, err error) {
	connType = RtmpPlay
	streamName = string(p.Stream)
	if !reflect.ValueOf(p.Duration).IsNil() {
		duration = float64(*p.Duration)
	}

	return
}

// the handler when got packet to identify the client.
// try next packet when loop is true and err is nil.
type rtmpIdentifyHandler func(p RtmpPacket) (loop bool, err error)

// identify the client.
func (v *RtmpConnection) identify(fn rtmpIdentifyHandler) (err error) {
	ctx := v.ctx

	err = v.read(IdentifyTimeout, func(m *RtmpMessage) (loop bool, err error) {
		var p RtmpPacket
		if p, err = v.stack.DecodeMessage(m); err != nil {
			return
		}

		// when parse got empty packet.
		if p == nil {
			core.Warn.Println(ctx, "ignore empty packet.")
			return true, nil
		}

		switch mt := p.MessageType(); mt {
		// ignore silently.
		case RtmpMsgAcknowledgement, RtmpMsgSetChunkSize, RtmpMsgWindowAcknowledgementSize, RtmpMsgUserControlMessage:
			return true, nil
		// matched
		case RtmpMsgAMF0CommandMessage, RtmpMsgAMF3CommandMessage:
			break
		// ignore with warning.
		default:
			core.Trace.Println(ctx, "ignore rtmp message", mt)
			return true, nil
		}

		// handler the packet which can identify the client.
		return fn(p)
	})
	return
}

// parse the rtmp packet to message.
func (v *RtmpConnection) packet2Message(p RtmpPacket, sid uint32) (m *RtmpMessage, err error) {
	m = NewRtmpMessage()

	var b bytes.Buffer
	if err = core.Marshal(p, &b); err != nil {
		return nil, err
	}

	m.MessageType = p.MessageType()
	m.PreferCid = p.PreferCid()
	m.StreamId = sid
	m.Payload = b.Bytes()

	return m, nil
}

// the handler when read a rtmp message.
// loop when loop is true and err is nil.
type rtmpReadHandler func(m *RtmpMessage) (loop bool, err error)

// read from cache and process by handler.
func (v *RtmpConnection) read(timeout time.Duration, fn rtmpReadHandler) (err error) {
	ctx := v.ctx

	for {
		select {
		case m := <-v.in:
			var loop bool
			if loop, err = fn(m); err != nil || !loop {
				return
			}
		case <-time.After(timeout):
			core.Error.Println(ctx, "timeout for", timeout)
			return core.TimeoutError
		case <-v.closing.QC():
			return v.closing.Quit()
		case <-v.wc.QC():
			return v.wc.Quit()
		}
	}

	return
}

// write to the cache.
func (v *RtmpConnection) write(p RtmpPacket, sid uint32) (err error) {
	var m *RtmpMessage
	if m, err = v.packet2Message(p, sid); err != nil {
		return
	}

	if err = v.CacheMessage(m); err != nil {
		return
	}

	return v.flush()
}

// receiver goroutine.
func (v *RtmpConnection) receiver() (err error) {
	ctx := v.ctx

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

		// push message to queue.
		// cache the message when got non empty one.
		select {
		case v.in <- m:
		case <-v.closing.QC():
			return v.closing.Quit()
		}
	}
	core.Warn.Println(ctx, "receiver ok.")

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

func (v RtmpMessageType) isData() bool {
	return v.isAmf0Data() || v.isAmf3Data()
}

func (v RtmpMessageType) IsCommand() bool {
	return v.isAmf0Command() || v.isAmf3Command()
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

	// FMLE
	Amf0CommandOnFcPublish   = "onFCPublish"
	Amf0CommandOnFcUnpublish = "onFCUnpublish"

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

// SoundFormat UB [4]
// Format of SoundData. The following values are defined:
//     0 = Linear PCM, platform endian
//     1 = ADPCM
//     2 = MP3
//     3 = Linear PCM, little endian
//     4 = Nellymoser 16 kHz mono
//     5 = Nellymoser 8 kHz mono
//     6 = Nellymoser
//     7 = G.711 A-law logarithmic PCM
//     8 = G.711 mu-law logarithmic PCM
//     9 = reserved
//     10 = AAC
//     11 = Speex
//     14 = MP3 8 kHz
//     15 = Device-specific sound
// Formats 7, 8, 14, and 15 are reserved.
// AAC is supported in Flash Player 9,0,115,0 and higher.
// Speex is supported in Flash Player 10 and higher.
type RtmpCodecAudio uint8

const (
	RtmpLinearPCMPlatformEndian RtmpCodecAudio = iota
	RtmpADPCM
	RtmpMP3
	RtmpLinearPCMLittleEndian
	RtmpNellymoser16kHzMono
	RtmpNellymoser8kHzMono
	RtmpNellymoser
	RtmpReservedG711AlawLogarithmicPCM
	RtmpReservedG711MuLawLogarithmicPCM
	RtmpReserved
	RtmpAAC
	RtmpSpeex
	RtmpReserved1CodecAudio
	RtmpReserved2CodecAudio
	RtmpReservedMP3_8kHz
	RtmpReservedDeviceSpecificSound
	RtmpReserved3CodecAudio
	RtmpDisabledCodecAudio
)

// AACPacketType IF SoundFormat == 10 UI8
// The following values are defined:
//     0 = AAC sequence header
//     1 = AAC raw
type RtmpAacType uint8

const (
	RtmpAacSequenceHeader RtmpAacType = iota
	RtmpAacRawData
	RtmpAacReserved
)

// E.4.3.1 VIDEODATA
// CodecID UB [4]
// Codec Identifier. The following values are defined:
//     2 = Sorenson H.263
//     3 = Screen video
//     4 = On2 VP6
//     5 = On2 VP6 with alpha channel
//     6 = Screen video version 2
//     7 = AVC
type RtmpCodecVideo uint8

const (
	RtmpReservedCodecVideo RtmpCodecVideo = iota
	RtmpReserved1CodecVideo
	RtmpSorensonH263
	RtmpScreenVideo
	RtmpOn2VP6
	RtmpOn2VP6WithAlphaChannel
	RtmpScreenVideoVersion2
	RtmpAVC
	RtmpDisabledCodecVideo
	RtmpReserved2CodecVideo
)

// E.4.3.1 VIDEODATA
// Frame Type UB [4]
// Type of video frame. The following values are defined:
//     1 = key frame (for AVC, a seekable frame)
//     2 = inter frame (for AVC, a non-seekable frame)
//     3 = disposable inter frame (H.263 only)
//     4 = generated key frame (reserved for server use only)
//     5 = video info/command frame
type RtmpAVCFrame uint8

const (
	RtmpReservedAVCFrame RtmpAVCFrame = iota
	RtmpKeyFrame
	RtmpInterFrame
	RtmpDisposableInterFrame
	RtmpGeneratedKeyFrame
	RtmpVideoInfoFrame
	RtmpReserved1AVCFrame
)

// AVCPacketType IF CodecID == 7 UI8
// The following values are defined:
//     0 = AVC sequence header
//     1 = AVC NALU
//     2 = AVC end of sequence (lower level NALU sequence ender is
//         not required or supported)
type RtmpVideoAVCType uint8

const (
	RtmpSequenceHeader RtmpVideoAVCType = iota
	RtmpNALU
	RtmpSequenceHeaderEOF
	RtmpReservedAVCType
)

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
	Payload []byte
}

func NewRtmpMessage() *RtmpMessage {
	return &RtmpMessage{}
}

func (v *RtmpMessage) String() string {
	return fmt.Sprintf("%v %vB %v", v.MessageType, len(v.Payload), v.Timestamp)
}

// convert the rtmp message to oryx message.
func (v *RtmpMessage) ToMessage() (core.Message, error) {
	return NewOryxRtmpMessage(v)
}

// covert the rtmp message to oryx message.
type OryxRtmpMessage struct {
	rtmp *RtmpMessage

	Metadata            bool
	VideoSequenceHeader bool
	AudioSequenceHeader bool
}

func NewOryxRtmpMessage(m *RtmpMessage) (*OryxRtmpMessage, error) {
	v := &OryxRtmpMessage{
		rtmp: m,
	}

	// whether sequence header.
	if v.rtmp.MessageType.isVideo() {
		v.VideoSequenceHeader = v.isVideoSequenceHeader()
	} else if v.rtmp.MessageType.isAudio() {
		v.AudioSequenceHeader = v.isAudioSequenceHeader()
	} else if v.rtmp.MessageType.isData() {
		// TODO: FIXME: implements it.
		v.Metadata = true
	}

	// parse the message, for example, decode the h.264 sps/pps.
	// TODO: FIXME: implements it.

	return v, nil
}

func (v *OryxRtmpMessage) isVideoSequenceHeader() bool {
	// TODO: FIXME: support other codecs.
	if len(v.rtmp.Payload) < 2 {
		return false
	}

	b := v.rtmp.Payload

	// sequence header only for h264
	codec := RtmpCodecVideo(b[0] & 0x0f)
	if codec != RtmpAVC {
		return false
	}

	frameType := RtmpAVCFrame((b[0] >> 4) & 0x0f)
	avcPacketType := RtmpVideoAVCType(b[1])
	return frameType == RtmpKeyFrame && avcPacketType == RtmpSequenceHeader
}

func (v *OryxRtmpMessage) isAudioSequenceHeader() bool {
	// TODO: FIXME: support other codecs.
	if len(v.rtmp.Payload) < 2 {
		return false
	}

	b := v.rtmp.Payload

	soundFormat := RtmpCodecAudio((b[0] >> 4) & 0x0f)
	if soundFormat != RtmpAAC {
		return false
	}

	aacPacketType := RtmpAacType(b[1])
	return aacPacketType == RtmpAacSequenceHeader
}

// copy the message headers, share body.
func (v *OryxRtmpMessage) Copy() *OryxRtmpMessage {
	mcp := *v.rtmp
	return &OryxRtmpMessage{
		rtmp: &mcp,
	}
}

func (v *OryxRtmpMessage) Timestamp() uint64 {
	return v.rtmp.Timestamp
}

func (v *OryxRtmpMessage) SetTimestamp(ts uint64) *OryxRtmpMessage {
	v.rtmp.Timestamp = ts
	return v
}

func (v *OryxRtmpMessage) Payload() *RtmpMessage {
	return v.rtmp
}

func (v *OryxRtmpMessage) String() string {
	return fmt.Sprintf("%v %vB", v.rtmp.MessageType, len(v.rtmp.Payload))
}

func (v *OryxRtmpMessage) Muxer() core.MessageMuxer {
	return core.MuxerRtmp
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

// 4.1.2. Call
// The call method of the NetConnection object runs remote procedure
// calls (RPC) at the receiving end. The called RPC name is passed as a
// parameter to the call command.
type RtmpCallPacket struct {
	// Name of the remote procedure that is called.
	Name Amf0String
	// If a response is expected we give a transaction Id. Else we pass a value of 0
	TransactionId Amf0Number
	// If there exists any command info this
	// is set, else this is set to null type.
	// @remark, optional, init to and maybe NULL.
	Command Amf0Any
	// Any optional arguments to be provided
	// @remark, optional, init to and maybe NULL.
	Args Amf0Any
}

func NewRtmpCallPacket() RtmpPacket {
	return &RtmpCallPacket{}
}

func (v *RtmpCallPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, v.Command, v.Args)
}

func (v *RtmpCallPacket) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)
	if err = core.Unmarshals(b, &v.Name, &v.TransactionId); err != nil {
		return
	}

	if b.Len() > 0 {
		if v.Command, err = Amf0Discovery(b.Bytes()); err != nil {
			return
		}
		if err = core.Unmarshals(b, v.Command); err != nil {
			return
		}
	}

	if b.Len() > 0 {
		if v.Args, err = Amf0Discovery(b.Bytes()); err != nil {
			return
		}
		if err = core.Unmarshals(b, v.Args); err != nil {
			return
		}
	}

	return
}

func (v *RtmpCallPacket) PreferCid() uint32 {
	return RtmpCidOverConnection
}

func (v *RtmpCallPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// response for RtmpCallPacket
type RtmpCallResPacket struct {
	// Name of the command.
	Name Amf0String
	// ID of the command, to which the response belongs to
	TransactionId Amf0Number
	// If there exists any command info this
	// is set, else this is set to null type.
	// @remark, optional, init to and maybe NULL.
	Command Amf0Any
	// Any optional arguments to be provided
	// @remark, optional, init to and maybe NULL.
	Args Amf0Any
}

func NewRtmpCallResPacket() RtmpPacket {
	return &RtmpCallResPacket{
		Name: Amf0String(Amf0CommandResult),
	}
}

func (v *RtmpCallResPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, v.Command, v.Args)
}

func (v *RtmpCallResPacket) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)
	if err = core.Unmarshals(b, &v.Name, &v.TransactionId); err != nil {
		return
	}

	if b.Len() > 0 {
		if v.Command, err = Amf0Discovery(b.Bytes()); err != nil {
			return
		}
		if err = core.Unmarshals(b, v.Command); err != nil {
			return
		}
	}

	if b.Len() > 0 {
		if v.Args, err = Amf0Discovery(b.Bytes()); err != nil {
			return
		}
		if err = core.Unmarshals(b, v.Args); err != nil {
			return
		}
	}

	return
}

func (v *RtmpCallResPacket) PreferCid() uint32 {
	return RtmpCidOverConnection
}

func (v *RtmpCallResPacket) MessageType() RtmpMessageType {
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

// response for RtmpCreateStreamPacket
type RtmpCreateStreamResPacket struct {
	// _result or _error; indicates whether the response is result or error.
	Name Amf0String
	// ID of the command that response belongs to.
	TransactionId Amf0Number
	// If there exists any command info this is set, else this is set to null type.
	Command Amf0Null
	// The return value is either a stream ID or an error information object.
	StreamId Amf0Number
}

func NewRtmpCreateStreamResPacket() RtmpPacket {
	return &RtmpCreateStreamResPacket{
		Name: Amf0String(Amf0CommandResult),
	}
}

func (v *RtmpCreateStreamResPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, &v.Command, &v.StreamId)
}

func (v *RtmpCreateStreamResPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Name, &v.TransactionId, &v.Command, &v.StreamId)
}

func (v *RtmpCreateStreamResPacket) PreferCid() uint32 {
	return RtmpCidOverConnection
}

func (v *RtmpCreateStreamResPacket) MessageType() RtmpMessageType {
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
	Stream Amf0String
}

func NewRtmpFMLEStartPacket() RtmpPacket {
	return &RtmpFMLEStartPacket{
		Name: Amf0String(Amf0CommandReleaseStream),
	}
}

func (v *RtmpFMLEStartPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, &v.Command, &v.Stream)
}

func (v *RtmpFMLEStartPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Name, &v.TransactionId, &v.Command, &v.Stream)
}

func (v *RtmpFMLEStartPacket) PreferCid() uint32 {
	return RtmpCidOverConnection
}

func (v *RtmpFMLEStartPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// response for RtmpFMLEStartPacket
type RtmpFMLEStartResPacket struct {
	// Name of the command
	Name Amf0String
	// the transaction ID to get the response.
	TransactionId Amf0Number
	// If there exists any command info this is set, else this is set to null type.
	// @remark, never be NULL, an AMF0 null instance.
	Command Amf0Null
	// the optional args, set to undefined.
	Args Amf0Undefined
}

func NewRtmpFMLEStartResPacket() RtmpPacket {
	return &RtmpFMLEStartResPacket{
		Name: Amf0String(Amf0CommandResult),
	}
}

func (v *RtmpFMLEStartResPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, &v.Command, &v.Args)
}

func (v *RtmpFMLEStartResPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Name, &v.TransactionId, &v.Command, &v.Args)
}

func (v *RtmpFMLEStartResPacket) PreferCid() uint32 {
	return RtmpCidOverConnection
}

func (v *RtmpFMLEStartResPacket) MessageType() RtmpMessageType {
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

// FMLE/flash publish
// 4.2.6. Publish
// The client sends the publish command to publish a named stream to the
// server. Using this name, any client can play this stream and receive
// the published audio, video, and data messages.
type RtmpPublishPacket struct {
	// Name of the command, set to "publish".
	Name Amf0String
	// Transaction ID set to 0.
	TransactionId Amf0Number
	// Command information object does not exist. Set to null type.
	Command Amf0Null
	// Name with which the stream is published.
	Stream Amf0String
	// Type of publishing. Set to "live", "record", or "append".
	//   record: The stream is published and the data is recorded to a new file.The file
	//           is stored on the server in a subdirectory within the directory that
	//           contains the server application. If the file already exists, it is
	//           overwritten.
	//   append: The stream is published and the data is appended to a file. If no file
	//           is found, it is created.
	//   live: Live data is published without recording it in a file.
	// @remark, SRS only support live.
	// @remark, optional, default to live.
	Type *Amf0String
}

func NewRtmpPublishPacket() RtmpPacket {
	return &RtmpPublishPacket{
		Name: Amf0String(Amf0CommandPublish),
	}
}

func (v *RtmpPublishPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, &v.Command, &v.Stream, v.Type)
}

func (v *RtmpPublishPacket) UnmarshalBinary(data []byte) (err error) {
	b := bytes.NewBuffer(data)
	if err = core.Unmarshals(b, &v.Name, &v.TransactionId, &v.Command, &v.Stream); err != nil {
		return
	}
	if b.Len() > 0 {
		v.Type = NewAmf0String("")
		return core.Unmarshals(b, v.Type)
	}
	return
}

func (v *RtmpPublishPacket) PreferCid() uint32 {
	return RtmpCidOverStream
}

func (v *RtmpPublishPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// onStatus command, AMF0 Call
// @remark, user must set the stream_id by SrsCommonMessage.set_packet().
type RtmpOnStatusCallPacket struct {
	// Name of command. Set to "onStatus"
	Name Amf0String
	// Transaction ID set to 0.
	TransactionId Amf0Number
	// Command information does not exist. Set to null type.
	Command Amf0Null
	// Name-value pairs that describe the response from the server.
	// 'code','level', 'description' are names of few among such information.
	Data *Amf0Object
}

func NewRtmpOnStatusCallPacket() RtmpPacket {
	return &RtmpOnStatusCallPacket{
		Name: Amf0String(Amf0CommandOnStatus),
		Data: NewAmf0Object(),
	}
}

func (v *RtmpOnStatusCallPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.TransactionId, &v.Command, v.Data)
}

func (v *RtmpOnStatusCallPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Name, &v.TransactionId, &v.Command, v.Data)
}

func (v *RtmpOnStatusCallPacket) PreferCid() uint32 {
	return RtmpCidOverStream
}

func (v *RtmpOnStatusCallPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0CommandMessage
}

// onStatus data, AMF0 Data
type RtmpOnStatusDataPacket struct {
	// Name of command. Set to "onStatus"
	Name Amf0String
	// Name-value pairs that describe the response from the server.
	// 'code', are names of few among such information.
	Data *Amf0Object
}

func NewRtmpOnStatusDataPacket() RtmpPacket {
	return &RtmpOnStatusDataPacket{
		Name: Amf0String(Amf0CommandOnStatus),
		Data: NewAmf0Object(),
	}
}

func (v *RtmpOnStatusDataPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, v.Data)
}

func (v *RtmpOnStatusDataPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Name, v.Data)
}

func (v *RtmpOnStatusDataPacket) PreferCid() uint32 {
	return RtmpCidOverStream
}

func (v *RtmpOnStatusDataPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0DataMessage
}

// AMF0Data RtmpSampleAccess
type RtmpSampleAccessPacket struct {
	// Name of command. Set to "|RtmpSampleAccess".
	Name Amf0String
	// whether allow access the sample of video.
	// @see: https://github.com/ossrs/srs/issues/49
	// @see: http://help.adobe.com/en_US/FlashPlatform/reference/actionscript/3/flash/net/NetStream.html#videoSampleAccess
	VideoSampleAccess Amf0Boolean
	// whether allow access the sample of audio.
	// @see: https://github.com/ossrs/srs/issues/49
	// @see: http://help.adobe.com/en_US/FlashPlatform/reference/actionscript/3/flash/net/NetStream.html#audioSampleAccess
	AudioSampleAccess Amf0Boolean
}

func NewRtmpSampleAccessPacket() RtmpPacket {
	return &RtmpSampleAccessPacket{
		Name: Amf0String(Amf0DataSampleAccess),
	}
}

func (v *RtmpSampleAccessPacket) MarshalBinary() (data []byte, err error) {
	return core.Marshals(&v.Name, &v.VideoSampleAccess, &v.AudioSampleAccess)
}

func (v *RtmpSampleAccessPacket) UnmarshalBinary(data []byte) (err error) {
	return core.Unmarshals(bytes.NewBuffer(data), &v.Name, &v.VideoSampleAccess, &v.AudioSampleAccess)
}

func (v *RtmpSampleAccessPacket) PreferCid() uint32 {
	return RtmpCidOverStream
}

func (v *RtmpSampleAccessPacket) MessageType() RtmpMessageType {
	return RtmpMsgAMF0DataMessage
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
	// the number of already received payload size.
	payload *bytes.Buffer

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
		payload: &bytes.Buffer{},
	}
}

// the interface for *bufio.Reader
type bufferedReader interface {
	io.Reader
	Peek(n int) ([]byte, error)
	ReadByte() (c byte, err error)
}

type debugBufferedReader struct {
	ctx core.Context
	imp *bufio.Reader
}

func NewReaderSize(ctx core.Context, r io.Reader, size int) bufferedReader {
	return &debugBufferedReader{
		ctx: ctx,
		imp: bufio.NewReaderSize(r, size),
	}
}

func (v *debugBufferedReader) Peek(n int) (b []byte, err error) {
	b, err = v.imp.Peek(n)
	return
}

func (v *debugBufferedReader) Read(p []byte) (n int, err error) {
	ctx := v.ctx

	if n, err = v.imp.Read(p); err != nil {
		return
	}

	first16B := 16
	if n < first16B {
		first16B = n
	}

	last16B := n - 16
	if last16B < 0 {
		last16B = 0
	}

	core.Trace.Println(ctx, fmt.Sprintf("Read p[%d] got %d, %#x, %#x", len(p), n, p[:first16B], p[last16B:n]))
	return
}

func (v *debugBufferedReader) ReadByte() (c byte, err error) {
	ctx := v.ctx

	if c, err = v.imp.ReadByte(); err != nil {
		return
	}

	core.Trace.Println(ctx, fmt.Sprintf("ReadByte got %#x", []byte{c}))
	return
}

// RTMP protocol stack.
type RtmpStack struct {
	ctx core.Context

	// the input and output stream.
	in  bufferedReader
	out io.Writer
	// the chunks for RTMP,
	// key is the cid from basic header.
	chunks map[uint32]*RtmpChunk
	// input chunk size, default to 128, set by peer packet.
	inChunkSize uint32
	// output chunk size, default to 128, set by peer packet.
	outChunkSize uint32

	// the protocol caches, for gc efficiency.
	// the chunk header c0, c3 and extended-timestamp caches.
	c0c3Cache [][]byte
	// the cache for the iovs.
	iovsCache [][]byte
	// use bufio instead byte buffer.
	slowSendBuffer *bufio.Writer
}

// max chunk header is fmt0.
const RtmpMaxChunkHeader = 12

// the preloaded group messages.
const RtmpDefaultMwMessages = 25

func NewRtmpStack(ctx core.Context, r io.Reader, w io.Writer) *RtmpStack {
	v := &RtmpStack{
		ctx:          ctx,
		out:          w,
		chunks:       make(map[uint32]*RtmpChunk),
		inChunkSize:  RtmpProtocolChunkSize,
		outChunkSize: RtmpProtocolChunkSize,
	}

	if core.Conf.Debug.RtmpDumpRecv {
		v.in = NewReaderSize(ctx, r, RtmpMaxChunkHeader)
	} else {
		v.in = bufio.NewReaderSize(r, RtmpMaxChunkHeader)
	}

	// assume each message contains 10 chunks,
	// and each chunk need 3 header(c0, c3, extended-timestamp).
	v.c0c3Cache = make([][]byte, RtmpDefaultMwMessages*10*3)
	for i := 0; i < len(v.c0c3Cache); i++ {
		v.c0c3Cache[i] = make([]byte, RtmpMaxChunkHeader)
	}
	// iovs cache contains the body cache.
	v.iovsCache = make([][]byte, 0, RtmpDefaultMwMessages*10*4)

	return v
}

func (v *RtmpStack) DecodeMessage(m *RtmpMessage) (p RtmpPacket, err error) {
	ctx := v.ctx
	b := bytes.NewBuffer(m.Payload)

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
			p = NewRtmpPublishPacket()
		case Amf0CommandOnFcPublish, "_checkbw":
			p = NewRtmpOnStatusCallPacket()
		// TODO: FIXME: implements it.
		default:
			core.Trace.Println(ctx, "drop command message, name is", c)
		}
	} else if m.MessageType.isUserControlMessage() {
		p = NewRtmpUserControlPacket()
	} else if m.MessageType.isWindowAckledgementSize() {
		p = NewRtmpSetWindowAckSizePacket()
	} else if m.MessageType.isSetChunkSize() {
		p = NewRtmpSetChunkSizePacket()
	} else {
		if !m.MessageType.isSetPeerBandwidth() && !m.MessageType.isAckledgement() {
			core.Trace.Println(ctx, "drop unknown message, type is", m.MessageType)
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
	ctx := v.ctx

	for m == nil {
		// chunk stream basic header.
		var fmt uint8
		var cid uint32
		if fmt, cid, err = rtmpReadBasicHeader(ctx, v.in); err != nil {
			if !core.IsNormalQuit(err) {
				core.Warn.Println(ctx, "read basic header failed. err is", err)
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
		if err = rtmpReadMessageHeader(ctx, v.in, fmt, chunk); err != nil {
			return
		}

		// read msg payload from chunk stream.
		if m, err = rtmpReadMessagePayload(ctx, v.inChunkSize, v.in, chunk); err != nil {
			return
		}
	}

	if err = v.onRecvMessage(m); err != nil {
		return nil, err
	}

	return
}

func (v *RtmpStack) onRecvMessage(m *RtmpMessage) (err error) {
	ctx := v.ctx

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
			core.Warn.Println(ctx, "accept invalid chunk size", p.ChunkSize)
		}
		v.inChunkSize = uint32(p.ChunkSize)
		core.Trace.Println(ctx, "input chunk size to", v.inChunkSize)
	case *RtmpUserControlPacket:
		// TODO: FIXME: implements it.
	}

	return
}

func (v *RtmpStack) onSendMessage(m *RtmpMessage) (err error) {
	ctx := v.ctx

	switch m.MessageType {
	case RtmpMsgSetChunkSize:
		// we will handle these packet.
	default:
		return
	}

	var p RtmpPacket
	if p, err = v.DecodeMessage(m); err != nil {
		return
	}

	switch p := p.(type) {
	case *RtmpSetChunkSizePacket:
		// for some server, the actual chunk size can greater than the max value(65536),
		// so we just warning the invalid chunk size, and actually use it is ok,
		// @see: https://github.com/ossrs/srs/issues/160
		if p.ChunkSize < RtmpMinChunkSize || p.ChunkSize > RtmpMaxChunkSize {
			core.Warn.Println(ctx, "accept invalid chunk size", p.ChunkSize)
		}
		v.outChunkSize = uint32(p.ChunkSize)
		core.Trace.Println(ctx, "output chunk size to", v.outChunkSize)
	}

	return
}

func (v *RtmpStack) fetchC0c3Cache(index int) (nextIndex int, iov []byte) {
	// exceed the index, create the cache.
	if index >= len(v.c0c3Cache) {
		iov = make([]byte, RtmpMaxChunkHeader)
		v.c0c3Cache = append(v.c0c3Cache, iov)
	}

	return index + 1, v.c0c3Cache[index]
}

// to sendout multiple messages.
func (v *RtmpStack) SendMessage(msgs ...*RtmpMessage) (err error) {
	// cache the messages to send to descrease the syscall.
	iovs := v.iovsCache

	var iovIndex int
	var iov []byte

	for _, m := range msgs {
		// we directly send out the packet,
		// use very simple algorithm, not very fast,
		// but it's ok.
		for written := uint32(0); written < uint32(len(m.Payload)); {
			// for chunk header without extended timestamp.
			if firstChunk := bool(written == 0); firstChunk {
				// the fmt0 is 12bytes header.
				iovIndex, iov = v.fetchC0c3Cache(iovIndex)
				iovs = append(iovs, iov[0:12])

				// write new chunk stream header, fmt is 0
				iov[0] = byte(0x00 | (byte(m.PreferCid) & 0x3f))

				// chunk message header, 11 bytes
				// timestamp, 3bytes, big-endian
				if m.Timestamp < RtmpExtendedTimestamp {
					iov[1] = byte(m.Timestamp >> 16)
					iov[2] = byte(m.Timestamp >> 8)
					iov[3] = byte(m.Timestamp)
				} else {
					iov[1] = 0xff
					iov[2] = 0xff
					iov[3] = 0xff
				}

				// message_length, 3bytes, big-endian
				iov[4] = byte(len(m.Payload) >> 16)
				iov[5] = byte(len(m.Payload) >> 8)
				iov[6] = byte(len(m.Payload))

				// message_type, 1bytes
				iov[7] = byte(m.MessageType)

				// stream_id, 4bytes, little-endian
				iov[8] = byte(m.StreamId)
				iov[9] = byte(m.StreamId >> 8)
				iov[10] = byte(m.StreamId >> 16)
				iov[11] = byte(m.StreamId >> 24)
			} else {
				// the fmt3 is 1bytes header.
				iovIndex, iov = v.fetchC0c3Cache(iovIndex)
				iovs = append(iovs, iov[0:1])

				// write no message header chunk stream, fmt is 3
				// @remark, if perfer_cid > 0x3F, that is, use 2B/3B chunk header,
				// SRS will rollback to 1B chunk header.
				iov[0] = byte(0xC0 | (byte(m.PreferCid) & 0x3f))
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
				// the extended-timestamp is 4bytes.
				iovIndex, iov = v.fetchC0c3Cache(iovIndex)
				iovs = append(iovs, iov[0:4])

				// big-endian.
				iov[0] = byte(len(m.Payload) >> 24)
				iov[1] = byte(len(m.Payload) >> 16)
				iov[2] = byte(len(m.Payload) >> 8)
				iov[3] = byte(len(m.Payload))
			}

			// write chunk payload
			var size uint32
			if size = uint32(len(m.Payload)) - written; size > v.outChunkSize {
				size = v.outChunkSize
			}
			iovs = append(iovs, m.Payload[written:written+size])

			written += size
		}

		if err = v.onSendMessage(m); err != nil {
			return
		}
	}

	//fmt.Println(ctx, fmt.Sprintf("fast send %v messages to %v iovecs", len(msgs), len(iovs)))
	return v.fastSendMessages(iovs...)
}

// group all messages to a big buffer and send it,
// for the os which not support writev.
// @remark this method will be invoked by platform depends fastSendMessages.
func (v *RtmpStack) slowSendMessages(iovs ...[]byte) (err error) {
	// delay init buffer.
	if v.slowSendBuffer == nil {
		// assume each message about 32kB
		v.slowSendBuffer = bufio.NewWriterSize(v.out, RtmpDefaultMwMessages*32*1024)
	}

	// write all pieces of iovs to buffer.
	for _, iov := range iovs {
		if _, err = v.slowSendBuffer.Write(iov); err != nil {
			return
		}
	}

	// send the buffer out.
	if err = v.slowSendBuffer.Flush(); err != nil {
		return
	}
	return
}

// read the RTMP message from buffer inb which load from reader in.
// return the completed message from chunk partial message.
func rtmpReadMessagePayload(ctx core.Context, chunkSize uint32, in bufferedReader, chunk *RtmpChunk) (m *RtmpMessage, err error) {
	m = chunk.partialMessage
	if m == nil {
		panic("chunk message should never be nil")
	}

	// the preload body must be consumed in a time.
	left := int(chunk.payloadLength) - chunk.payload.Len()
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
	if _, err = io.CopyN(chunk.payload, in, int64(left)); err != nil {
		core.Error.Println(ctx, "read body failed. err is", err)
		return
	}

	// got entire RTMP message?
	if int(chunk.payloadLength) == chunk.payload.Len() {
		chunk.partialMessage.Payload = chunk.payload.Bytes()
		chunk.payload = &bytes.Buffer{}
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
func rtmpReadMessageHeader(ctx core.Context, in bufferedReader, fmt uint8, chunk *RtmpChunk) (err error) {
	if core.Conf.Debug.RtmpDumpRecv {
		core.Trace.Println(ctx, "start read message header.")
	}

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
			core.Warn.Println(ctx, "accept cid=2,fmt=1 to make librtmp happy.")
		} else {
			// must be a RTMP protocol level error.
			core.Error.Println(ctx, "fresh chunk fmt must be", RtmpFmtType0, "actual is", fmt)
			return RtmpChunkError
		}
	}

	// when exists cache msg, means got an partial message,
	// the fmt must not be type0 which means new message.
	if !isFirstMsgOfChunk && fmt == RtmpFmtType0 {
		core.Error.Println(ctx, "chunk partial msg, fmt must be", RtmpFmtType0, "actual is", fmt)
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
		if bh, err = in.Peek(nbh); err != nil {
			return
		}
		if _, err = io.CopyN(ioutil.Discard, in, int64(nbh)); err != nil {
			return
		}
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
			core.Error.Println(ctx, "chunk msg exists, should not change the delta.")
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
				core.Error.Println(ctx, "chunk msg exists, payload length should not be changed.")
				return RtmpChunkError
			}
			// for a message, if msg exists in cache, the type must not changed.
			if !isFirstMsgOfChunk && chunk.messageType != mtype {
				core.Error.Println(ctx, "chunk msg exists, type should not be changed.")
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
		var b []byte
		if b, err = in.Peek(4); err != nil {
			return
		}

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
			if core.Conf.Debug.RtmpDumpRecv {
				core.Trace.Println(ctx, "read matched extended-timestamp.")
			}
			// consume from buffer.
			if _, err = io.CopyN(ioutil.Discard, in, 4); err != nil {
				return
			}
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
func rtmpReadBasicHeader(ctx core.Context, in bufferedReader) (fmt uint8, cid uint32, err error) {
	if core.Conf.Debug.RtmpDumpRecv {
		core.Trace.Println(ctx, "start read basic header.")
	}

	var vb byte
	if vb, err = in.ReadByte(); err != nil {
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
		if vb, err = in.ReadByte(); err != nil {
			return
		}

		temp := uint32(vb) + 64

		// 64-65599, 3B chunk header
		if cid >= 1 {
			if vb, err = in.ReadByte(); err != nil {
				return
			}

			temp += uint32(vb) * 256
		}

		return fmt, temp, nil
	}

	return fmt, cid, RtmpChunkError
}
