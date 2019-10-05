/*
The MIT License (MIT)

Copyright (c) 2016 winlin

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

/*
 This is the listeners for oryx.
*/
package main

import (
	"fmt"
	ol "github.com/ossrs/go-oryx-lib/logger"
	"io"
	"net"
	"strings"
	"sync"
	"context"
)

// The tcp listeners which support reload.
type TcpListeners struct {
	// The config and listener objects.
	addrs     []string
	listeners []*net.TCPListener
	// Used to get the connection or error for accept.
	conns  chan *net.TCPConn
	errors chan error
	// Used to ensure all gorutine quit.
	wait *sync.WaitGroup
	// Used to notify all goroutines to quit.
	// TODO: FIXME: Use context.Context instead.
	closing chan bool
	closed  bool
}

// Listen at addrs format as netowrk://laddr, for example,
// tcp://:1935, tcp4://:1935, tcp6://1935, tcp://0.0.0.0:1935
func NewTcpListeners(addrs []string) (v *TcpListeners, err error) {
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no listens")
	}

	for _, v := range addrs {
		if !strings.HasPrefix(v, "tcp://") && !strings.HasPrefix(v, "tcp4://") && !strings.HasPrefix(v, "tcp6://") {
			return nil, fmt.Errorf("%v should prefix with tcp://, tcp4:// or tcp6://", v)
		}
		if n := strings.Count(v, "://"); n != 1 {
			return nil, fmt.Errorf("%v contains %d network identify", v, n)
		}
	}

	v = &TcpListeners{
		addrs:   addrs,
		conns:   make(chan *net.TCPConn),
		errors:  make(chan error),
		wait:    &sync.WaitGroup{},
		closing: make(chan bool, 1),
	}

	return
}

func (v *TcpListeners) ListenTCP(ctx context.Context) (err error) {
	for _, addr := range v.addrs {
		var network, laddr string
		if vs := strings.Split(addr, "://"); true {
			network, laddr = vs[0], vs[1]
		}

		var l net.Listener
		if l, err = net.Listen(network, laddr); err != nil {
			return
		} else if l, ok := l.(*net.TCPListener); !ok {
			panic("listener: must be *net.TCPListener")
		} else {
			v.listeners = append(v.listeners, l)
		}
	}

	for i, l := range v.listeners {
		go v.acceptFrom(ctx, l, v.addrs[i])
	}

	return
}

func (v *TcpListeners) acceptFrom(ctx context.Context, l *net.TCPListener, addr string) {
	v.wait.Add(1)
	defer v.wait.Done()

	for {
		if err := v.doAcceptFrom(ctx, l); err != nil {
			if err != io.EOF {
				ol.W(ctx, "listener:", addr, "quit, err is", err)
			}
			return
		}
	}

	return
}

func (v *TcpListeners) doAcceptFrom(ctx context.Context, l *net.TCPListener) (err error) {
	defer func() {
		if err != nil && err != io.EOF {
			select {
			case v.errors <- err:
			case c := <-v.closing:
				v.closing <- c
			}
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			if err != nil {
				ol.E(ctx, "listener: recover from", r, "and err is", err)
				return
			}

			if r, ok := r.(error); ok {
				err = r
			} else {
				err = fmt.Errorf("system error", r)
			}
			ol.E(ctx, "listener: recover from", err)
		}
	}()

	var conn *net.TCPConn
	if conn, err = l.AcceptTCP(); err != nil {
		// when disposed, ignore any error for it's user closed listener.
		select {
		case c := <-v.closing:
			err = io.EOF
			v.closing <- c
		default:
			ol.E(ctx, "listener: accept failed, err is", err)
		}
		return
	}

	select {
	case v.conns <- conn:
	case c := <-v.closing:
		v.closing <- c

		// we got a connection but not accept by user and listener is closed,
		// we must close this connection for user never get it.
		conn.Close()
		ol.W(ctx, "listener: drop connection", conn.RemoteAddr())
	}

	return
}

// @remark when user closed the listener, err is io.EOF.
func (v *TcpListeners) AcceptTCP() (c *net.TCPConn, err error) {
	var ok bool
	select {
	case c, ok = <-v.conns:
	case err, ok = <-v.errors:
	case c := <-v.closing:
		v.closing <- c
		return nil, io.EOF
	}

	// when chan closed, the listener is disposed.
	if !ok {
		return nil, io.EOF
	}
	return
}

// io.Closer
// User should never reuse the closed instance.
func (v *TcpListeners) Close() (err error) {
	if v.closed {
		return
	}
	v.closed = true

	// unblock all listener and user goroutines
	select {
	case v.closing <- true:
	default:
	}

	// interrupt all listeners.
	for _, v := range v.listeners {
		if r := v.Close(); r != nil {
			err = r
		}
	}

	// wait for all listener internal goroutines to quit.
	v.wait.Wait()

	// close channels to unblock the user goroutine to AcceptTCP()
	close(v.conns)
	close(v.errors)

	return
}
