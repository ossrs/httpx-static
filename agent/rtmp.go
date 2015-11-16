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
	"net"
	"fmt"
)

// the rtmp publish agent,
// to listen at RTMP(tcp://1935) and recv data from RTMP publisher,
// for example, the FMLE publisher.
type RtmpPublish struct {
	l net.Listener
}

func NewRtmpPublish(wc core.WorkerContainer) (agent core.Agent, err error) {
	r := &RtmpPublish{}

	ep := fmt.Sprintf(":%v", core.Conf.Listen)
	if r.l,err = net.Listen("tcp", ep); err != nil {
		core.Error.Println("rtmp listen at", ep, "failed. err is", err)
		return
	}
	core.Trace.Println("rtmp listen at", ep)

	return r,nil
}

// interface core.Agent
func (v *RtmpPublish) Source() core.Source {
	return nil
}

func (v *RtmpPublish) Channel() chan *core.Message {
	return nil
}

func (v *RtmpPublish) Sink() core.Sink {
	return nil
}
