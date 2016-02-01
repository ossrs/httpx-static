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

// +build darwin dragonfly freebsd nacl netbsd openbsd solaris linux

// Unix reload by signal.

package core

import (
	"os"
	"os/signal"
	"syscall"
)

func (v *Config) ReloadCycle(wc WorkerContainer) {
	ctx := v.ctx

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP)

	Trace.Println(ctx, "wait for reload signals: kill -1", os.Getpid())
	for {
		select {
		case signal := <-signals:
			Trace.Println(ctx, "start reload by", signal)

			if err := v.doReload(); err != nil {
				defer wc.Quit()
				Error.Println(ctx, "quit for reload failed. err is", err)
				return
			}

		case <-wc.QC():
			defer wc.Quit()
			Warn.Println(ctx, "user stop server")
			return
		}
	}
}

func (v *Config) doReload() (err error) {
	ctx := v.ctx

	pc := v
	cc := NewConfig(v.ctx)
	cc.reloadHandlers = pc.reloadHandlers[:]
	if err = cc.Loads(v.conf); err != nil {
		Error.Println(ctx, "reload config failed. err is", err)
		return
	}
	Info.Println(ctx, "reload parse fresh config ok")

	if err = pc.Reload(cc); err != nil {
		Error.Println(ctx, "apply reload failed. err is", err)
		return
	}
	Info.Println(ctx, "reload completed work")

	Conf = cc
	Trace.Println(ctx, "reload config ok")

	return
}
