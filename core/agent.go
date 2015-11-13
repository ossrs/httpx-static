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

// the message for oryx
// the common structure for RTMP/FLV/HLS/MP4 or any
// message, it can be media message or control message.
// the message flow from agent to another agent.
type Message struct {
}

// the source of agent,
// to ingest message from upstream sink
// then produce to channel.
type Source interface {
}

// the sink of agent,
// to consume message from channel
// then delivery to downstream source.
type Sink interface {
}

// the agent contains a source
// which ingest message from upstream sink
// write message to channel
// finally delivery to downstream sink.
type Agent interface {
	// the source of agent.
	Source() Source
	// the channel of agent.
	Channel() chan *Message
	// the sink of agent.
	Sink() Sink

	// tie the current to agent
	// where agent is the upstream of current
	// and current is the downstream of agent.
	// 		agent.Sink => current.Source
	Tie(agent Agent) (err error)
	// flow the current to agent
	// where current is the upstream of agent
	// and agent is the downstream of current.
	//		current.Sink => agent.Source
	Flow(agent Agent) (err error)
}
