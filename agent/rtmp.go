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
	"fmt"
	"github.com/ossrs/go-oryx/core"
	"net"
)

// the rtmp publish agent,
// to listen at RTMP(tcp://1935) and recv data from RTMP publisher,
// for example, the FMLE publisher.
type RtmpPublish struct {
	endpoint string
	wc       core.WorkerContainer
	l        net.Listener
}

func NewRtmpPublish(wc core.WorkerContainer) (agent core.Agent) {
	v := &RtmpPublish{
		wc: wc,
	}

	core.Conf.Subscribe(v)

	return v
}

// interface core.Agent
func (v *RtmpPublish) Open() (err error) {
	return v.applyListen(core.Conf)
}

func (v *RtmpPublish) Close() (err error) {
	core.Conf.Unsubscribe(v)
	return v.close()
}

func (v *RtmpPublish) Source() (ss core.Source) {
	return nil
}

func (v *RtmpPublish) Channel() (c chan *core.Message) {
	return nil
}

func (v *RtmpPublish) Sink() (sk core.Sink) {
	return nil
}

func (v *RtmpPublish) close() (err error) {
	if v.l == nil {
		return
	}

	if err = v.l.Close(); err != nil {
		core.Error.Println("close rtmp listener failed. err is", err)
		return
	}

	core.Trace.Println("close rtmp listen", v.endpoint, "ok")
	return
}

func (v *RtmpPublish) applyListen(c *core.Config) (err error) {
	v.endpoint = fmt.Sprintf(":%v", c.Listen)

	ep := v.endpoint
	if v.l, err = net.Listen("tcp", ep); err != nil {
		core.Error.Println("rtmp listen at", ep, "failed. err is", err)
		return
	}
	core.Trace.Println("rtmp listen at", ep)

	return
}

// interface ReloadHandler
func (v *RtmpPublish) OnReloadGlobal(scope int, cc, pc *core.Config) (err error) {
	if scope != core.ReloadListen {
		return
	}

	if err = v.close(); err != nil {
		return
	}

	if err = v.applyListen(cc); err != nil {
		return
	}

	return
}
