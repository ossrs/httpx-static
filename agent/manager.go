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

package agent

import (
	"github.com/ossrs/go-oryx/core"
	"github.com/ossrs/go-oryx/protocol"
	"sync"
)

type AgentManager struct {
	// the media stream sources agent,
	// generally it's a dup agent.
	sources map[string]core.Agent
	lock    sync.Mutex
}

var Manager *AgentManager = NewManager()

func NewManager() *AgentManager {
	return &AgentManager{
		sources: make(map[string]core.Agent),
	}
}

func (v *AgentManager) Close() {
	for _, v := range v.sources {
		if err := v.Close(); err != nil {
			core.Warn.Println("ignore close agent failed. err is", err)
		}
	}
}

func (v *AgentManager) NewRtmpPlayAgent(conn *protocol.RtmpConnection, wc core.WorkerContainer) (a core.Agent, err error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	// finger the source agent out, which dup to other agent.
	var dup core.Agent
	if dup, err = v.getDupAgent(conn.Req.Uri()); err != nil {
		return
	}

	// create the publish agent
	a = &RtmpPlayAgent{
		conn: conn,
		wc:   wc,
	}

	// tie the play agent to dup sink.
	if err = a.Tie(dup); err != nil {
		core.Error.Println("tie agent failed. err is", err)
		return
	}

	return
}

func (v *AgentManager) NewRtmpPublishAgent(conn *protocol.RtmpConnection, wc core.WorkerContainer) (a core.Agent, err error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	// finger the source agent out, which dup to other agent.
	var dup core.Agent
	if dup, err = v.getDupAgent(conn.Req.Uri()); err != nil {
		return
	}

	// when dup source not nil, then the source is using.
	if dup.TiedSink() != nil {
		err = AgentBusyError
		core.Error.Println("source busy. err is", err)
		return
	}

	// create the publish agent
	a = &RtmpPublishAgent{
		conn: conn,
		wc:   wc,
	}

	// tie the publish agent to dup source.
	if err = dup.Tie(a); err != nil {
		core.Error.Println("tie agent failed. err is", err)
		return
	}

	return
}

func (v *AgentManager) getDupAgent(uri string) (dup core.Agent, err error) {
	var ok bool
	if dup, ok = v.sources[uri]; !ok {
		dup = NewDupAgent()
		v.sources[uri] = dup

		if err = dup.Open(); err != nil {
			core.Error.Println("open dup agent failed. err is", err)
			return
		}

		// start async work for dup worker.
		wait := make(chan bool, 1)
		core.Recover("", func() (err error) {
			wait <- true

			if err = dup.Pump(); err != nil {
				core.Error.Println("dup agent work failed. err is", err)
				return
			}
			return
		})
		<-wait
	}

	return
}
