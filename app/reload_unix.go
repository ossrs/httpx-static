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

// +build darwin dragonfly freebsd nacl netbsd openbsd solaris linux

// Unix reload by signal.

package app

import (
	"github.com/ossrs/go-oryx/core"
	"os"
	"os/signal"
	"syscall"
)

func (c *Config) reloadCycle(wc WorkerContainer) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP)

	core.GsTrace.Println("wait for reload signals: kill -1", os.Getpid())
	for {
		select {
		case signal := <-signals:
			core.GsTrace.Println("start reload by", signal)

			if err := c.doReload(); err != nil {
				core.GsError.Println("quit for reload failed. err is", err)
				wc.Quit()
				return
			}

		case <-wc.QC():
			core.GsWarn.Println("user stop reload")
			wc.Quit()
			return
		}
	}
}

func (c *Config) doReload() (err error) {
	pc := c
	cc := NewConfig()
	cc.reloadHandlers = pc.reloadHandlers[:]
	if err = cc.Loads(c.conf); err != nil {
		core.GsError.Println("reload config failed. err is", err)
		return
	}
	core.GsInfo.Println("reload parse fresh config ok")

	if err = pc.Reload(cc); err != nil {
		core.GsError.Println("apply reload failed. err is", err)
		return
	}
	core.GsInfo.Println("reload completed work")

	GsConfig = cc
	core.GsTrace.Println("reload config ok")

	return
}
