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
	"github.com/ossrs/go-oryx/protocol"
	"net"
	"runtime/debug"
)

// the rtmp publish or play agent,
// to listen at RTMP(tcp://1935) and recv data from RTMP publisher or player,
// when identified the client type, redirect to the specified agent.
type Rtmp struct {
	sid      uint32
	endpoint string
	wc       core.WorkerContainer
	l        net.Listener
}

func NewRtmp(wc core.WorkerContainer) (agent core.OpenCloser) {
	v := &Rtmp{
		sid: 1,
		wc:  wc,
	}

	core.Conf.Subscribe(v)

	return v
}

// interface core.Agent
func (v *Rtmp) Open() (err error) {
	return v.applyListen(core.Conf)
}

func (v *Rtmp) Close() (err error) {
	core.Conf.Unsubscribe(v)
	return v.close()
}

func (v *Rtmp) close() (err error) {
	if v.l == nil {
		return
	}

	if err = v.l.Close(); err != nil {
		core.Error.Println("close rtmp listener failed. err is", err)
		return
	}
	v.l = nil

	core.Trace.Println("close rtmp listen", v.endpoint, "ok")
	return
}

func (v *Rtmp) applyListen(c *core.Config) (err error) {
	v.endpoint = fmt.Sprintf(":%v", c.Listen)

	ep := v.endpoint
	if v.l, err = net.Listen("tcp", ep); err != nil {
		core.Error.Println("rtmp listen at", ep, "failed. err is", err)
		return
	}
	core.Trace.Println("rtmp listen at", ep)

	// accept cycle
	v.wc.GFork("", func(wc core.WorkerContainer) {
		for v.l != nil {
			var c net.Conn
			if c, err = v.l.Accept(); err != nil {
				if v.l != nil {
					core.Warn.Println("accept failed. err is", err)
				}
				return
			}

			// use gfork to serve the connection.
			v.wc.GFork("", func(wc core.WorkerContainer) {
				defer func() {
					if r := recover(); r != nil {
						if !core.IsNormalQuit(r) {
							core.Warn.Println("rtmp ignore", r)
						}

						core.Error.Println(string(debug.Stack()))
					}
				}()

				conn, err := v.identify(c)
				defer conn.Close()

				if !core.IsNormalQuit(err) {
					core.Warn.Println("ignore error when identify rtmp. err is", err)
					return
				}
				core.Info.Println("rtmp identify ok.")
			})
		}
	})

	// should quit?
	v.wc.GFork("", func(wc core.WorkerContainer) {
		<-wc.QC()
		_ = v.close()
		wc.Quit()
	})

	return
}

func (v *Rtmp) identify(c net.Conn) (conn *protocol.RtmpConnection, err error) {
	conn = protocol.NewRtmpConnection(c, v.wc)

	core.Trace.Println("rtmp accept", c.RemoteAddr())

	// handshake with client.
	if err = conn.Handshake(); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println("rtmp handshake failed. err is", err)
		}
		return
	}
	core.Info.Println("rtmp handshake ok.")

	// expoect connect app.
	r := protocol.NewRtmpRequest()
	if err = conn.ExpectConnectApp(r); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println("rtmp connnect app failed. err is", err)
		}
		return
	}
	core.Info.Println("rtmp connect app ok, tcUrl is", r.TcUrl)

	if err = conn.SetWindowAckSize(uint32(2.5 * 1000 * 1000)); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println("rtmp set ack size failed. err is", err)
		}
		return
	}

	if err = conn.SetPeerBandwidth(uint32(2.5*1000*1000), uint8(2)); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println("rtmp set peer bandwidth failed. err is", err)
		}
		return
	}

	// do bandwidth test if connect to the vhost which is for bandwidth check.
	// TODO: FIXME: support bandwidth check.

	// do token traverse before serve it.
	// @see https://github.com/ossrs/srs/pull/239
	// TODO: FIXME: support edge token tranverse.

	// set chunk size to larger.
	// set the chunk size before any larger response greater than 128,
	// to make OBS happy, @see https://github.com/ossrs/srs/issues/454
	// TODO: FIXME: support set chunk size.

	// response the client connect ok and onBWDone.
	if err = conn.ResponseConnectApp(); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println("response connect app failed. err is", err)
		}
		return
	}
	if err = conn.OnBwDone(); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println("response onBWDone failed. err is", err)
		}
		return
	}

	// increasing the stream id.
	v.sid++

	// identify the client, publish or play.
	if r.Type, r.Stream, r.Duration, err = conn.Identify(v.sid); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println("identify client failed. err is", err)
		}
		return
	}
	core.Trace.Println(fmt.Sprintf(
		"client identified, type=%s, stream_name=%s, duration=%.2f",
		r.Type, r.Stream, r.Duration))

	// reparse the request by connect and play/publish.
	if err = r.Reparse(); err != nil {
		core.Error.Println("reparse request failed. err is", err)
		return
	}

	// security check
	// TODO: FIXME: implements it.

	// set the TCP_NODELAY to false for high performance.
	// or set tot true for realtime stream.
	// TODO: FIXME: implements it.

	// check vhost.
	// for standard rtmp, the vhost specified in connectApp(tcUrl),
	// while some new client specifies the vhost in stream.
	// for example,
	//		connect("rtmp://vhost/app"), specified in tcUrl.
	//		connect("rtmp://ip/app?vhost=vhost"), specified in tcUrl.
	//		connect("rtmp://ip/app") && play("stream?vhost=vhost"), specified in stream.
	var vhost *core.Vhost
	if vhost, err = core.Conf.Vhost(r.Vhost); err != nil {
		core.Error.Println("check vhost failed, vhost is", r.Vhost, "and err is", err)
		return
	} else if r.Vhost != vhost.Name {
		core.Trace.Println("redirect vhost", r.Vhost, "to", vhost.Name)
		r.Vhost = vhost.Name
	}

	return
}

// interface ReloadHandler
func (v *Rtmp) OnReloadGlobal(scope int, cc, pc *core.Config) (err error) {
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
