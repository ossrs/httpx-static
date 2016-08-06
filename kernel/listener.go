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
package kernel

import (
	"fmt"
	ol "github.com/ossrs/go-oryx-lib/logger"
	"net"
	"strings"
	"sync"
)

// The listener is disposed by user.
var ListenerDisposed error = fmt.Errorf("listener disposed")

// The tcp listeners which support reload.
// @remark listener will return error ListenerDisposed when reuse a disposed listener.
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
	closing chan bool
	// Used to prevent reuse this object.
	disposed  bool
	reuseLock *sync.Mutex
}

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
		addrs:     addrs,
		conns:     make(chan *net.TCPConn),
		errors:    make(chan error),
		wait:      &sync.WaitGroup{},
		closing:   make(chan bool, 1),
		reuseLock: &sync.Mutex{},
	}

	return
}

// @remark error ListenerDisposed when listener is disposed.
func (v *TcpListeners) ListenTCP() (err error) {
	if err = func() error {
		v.reuseLock.Lock()
		defer v.reuseLock.Unlock()

		// user should never listen on a disposed listener
		if v.disposed {
			return ListenerDisposed
		}
		return nil
	}(); err != nil {
		return
	}

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

	if len(v.listeners) > 0 {
		v.wait.Add(len(v.listeners))
	}

	for i, l := range v.listeners {
		go func(l *net.TCPListener) {
			defer v.wait.Done()

			ctx := &Context{}
			addr := v.addrs[i]

			for {
				if err := v.acceptFrom(ctx, l); err != nil {
					if err != ListenerDisposed {
						ol.W(ctx, "listener:", addr, "quit, err is", err)
					}
					return
				}
			}
		}(l)
	}

	return
}

func (v *TcpListeners) acceptFrom(ctx ol.Context, l *net.TCPListener) (err error) {
	defer func() {
		if err != nil && err != ListenerDisposed {
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
		// when disposed, ignore any error.
		if v.disposed {
			err = ListenerDisposed
			return
		}

		ol.E(ctx, "listener: accept failed, err is", err)
		return
	}

	select {
	case v.conns <- conn:
	case c := <-v.closing:
		v.closing <- c
	}

	return
}

// @remark error ListenerDisposed when listener is disposed.
func (v *TcpListeners) AcceptTCP() (c *net.TCPConn, err error) {
	if err = func() error {
		v.reuseLock.Lock()
		defer v.reuseLock.Unlock()

		// user should never use a disposed listener.
		if v.disposed {
			return ListenerDisposed
		}
		return nil
	}(); err != nil {
		return nil, err
	}

	var ok bool
	select {
	case c, ok = <-v.conns:
	case err, ok = <-v.errors:
	}

	// when chan closed, the listener is disposed.
	if !ok {
		return nil, ListenerDisposed
	}
	return
}

// io.Closer
// @remark error ListenerDisposed when listener is disposed.
func (v *TcpListeners) Close() (err error) {
	if err = func() error {
		v.reuseLock.Lock()
		defer v.reuseLock.Unlock()

		// user should close a disposed listener.
		if v.disposed {
			return ListenerDisposed
		}

		// set to disposed to prevent reuse this object.
		v.disposed = true
		return nil
	}(); err != nil {
		return
	}

	// unblock all goroutines
	v.closing <- true

	// interrupt all listeners.
	for _, v := range v.listeners {
		if r := v.Close(); r != nil {
			err = r
		}
	}

	// wait for all listener to quit.
	v.wait.Wait()

	// clear the closing signal.
	_ = <-v.closing

	// close channels to unblock the user goroutine to AcceptTCP()
	close(v.conns)
	close(v.errors)

	return
}
