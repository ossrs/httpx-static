// The MIT License (MIT)
//
// Copyright (c) 2013-2016 Oryx(ossrs)
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
	"fmt"
	"io"
)

// MessageMuxer is the muxer of oryx's message type.
type MessageMuxer uint8

const (
	// MuxerRtmp represents the RTMP Muxer
	MuxerRtmp MessageMuxer = iota
	// MuxerFlv represents the FLV Muxer
	MuxerFlv
	// MuxerH264 represents the H264 Muxer
	MuxerH264
	// MuxerRtsp represents the RTSP Muxer
	MuxerRtsp
	// MuxerTs represents the TS Muxer
	MuxerTs
	// MuxerAac represents the AAC Muxer
	MuxerAac
	// MuxerMp3 represents the MP3 Muxer
	MuxerMp3
)

// Message for oryx
// the common structure for RTMP/FLV/HLS/MP4 or any
// message, it can be media message or control message.
// the message flow from agent to another agent.
type Message interface {
	fmt.Stringer

	// the muxer of message.
	Muxer() MessageMuxer
}

// Opener to open the resource.
type Opener interface {
	// open the resource.
	Open() error
}

// OpenCloser is the open and closer for resource management.
type OpenCloser interface {
	Opener
	io.Closer
}

// Agent contains a source
// which ingest message from upstream sink
// write message to channel
// finally delivery to downstream sink.
//
// the arch for agent is:
//      +-----upstream----+           +---downstream----+
//    --+-source => sink--+--(tie->)--+-source => sink--+--
//      +-----------------+           +-----------------+
//
// @remark all method is sync, user should never assume it's async.
type Agent interface {
	// an agent is a resource manager.
	OpenCloser

	// do agent jobs, to pump messages
	// from source to sink.
	Pump() (err error)
	// write to source, from upstream sink.
	Write(m Message) (err error)

	// source tie to the upstream sink.
	Tie(sink Agent) (err error)
	// destroy the link between source and upstream sink.
	UnTie(sink Agent) (err error)
	// get the tied upstream sink of source.
	TiedSink() (sink Agent)

	// sink flow to the downstream source.
	// @remark internal api, sink.Flow(source) when source.tie(sink).
	Flow(source Agent) (err error)
	// destroy the link between sink and downstream sink.
	UnFlow(source Agent) (err error)
}
