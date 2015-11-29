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
)

type DupAgent struct {
	upstream core.Agent
	sources  []core.Agent
}

func NewDupAgent() core.Agent {
	return &DupAgent{
		sources: make([]core.Agent, 0),
	}
}

func (v *DupAgent) Open() (err error) {
	return
}

func (v *DupAgent) Close() (err error) {
	return
}

func (v *DupAgent) Pump() (err error) {
	core.Error.Println("dup agent not support pump.")
	return AgentNotSupportError
}

func (v *DupAgent) Write(m core.Message) (err error) {
	for _, a := range v.sources {
		if err = a.Write(m); err != nil {
			return
		}
	}

	return
}

func (v *DupAgent) Tie(sink core.Agent) (err error) {
	v.upstream = sink
	return v.upstream.Flow(v)
}

func (v *DupAgent) UnTie(sink core.Agent) (err error) {
	v.upstream = nil
	return sink.UnFlow(v)
}

func (v *DupAgent) Flow(source core.Agent) (err error) {
	v.sources = append(v.sources, source)
	return
}

func (v *DupAgent) UnFlow(source core.Agent) (err error) {
	for i, s := range v.sources {
		if s == source {
			v.sources = append(v.sources[:i], v.sources[i+1:]...)
			break
		}
	}
	return
}

func (v *DupAgent) TiedSink() (sink core.Agent) {
	return v.upstream
}
