// The MIT License (MIT)
//
// Copyright (c) 2013-2015 SRS(simple-rtmp-server)
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

// The os defines:
//      bsd: darwin dragonfly freebsd nacl netbsd openbsd solaris
//      unix: bsd linux
//      server: unix plan9
//      posix: bsd linux windows
// All os by go:
//      server windows
//      posix plan9

package main

import (
	"flag"
	"fmt"
	"github.com/ossrs/go-daemon"
	"github.com/ossrs/go-srs/app"
	"github.com/ossrs/go-srs/core"
	"os"
)

// the startup argv:
//          -c conf/srs.json
//          --c conf/srs.json
//          -c=conf/srs.json
//          --c=conf/srs.json
var confFile = flag.String("c", "conf/srs.json", "the config file.")

func serve(svr *app.Server) int {
	if err := svr.PrepareLogger(); err != nil {
		core.GsError.Println("prepare logger failed, err is", err)
		return -1
	}

	core.GsTrace.Println("SRS start serve, pid is", os.Getpid(), "and ppid is", os.Getppid())
	core.GsTrace.Println("Copyright (c) 2013-2015 SRS(simple-rtmp-server)")
	core.GsTrace.Println(fmt.Sprintf("GO-SRS/%v is a golang implementation of SRS.", core.Version()))

	if err := svr.Initialize(); err != nil {
		core.GsError.Println("initialize server failed, err is", err)
		return -1
	}

	if err := svr.Run(); err != nil {
		core.GsError.Println("run server failed, err is", err)
		return -1
	}

	return 0
}

func run() int {
	flag.Parse()

	svr := app.NewServer()
	defer svr.Close()

	if err := svr.ParseConfig(*confFile); err != nil {
		core.GsError.Println("parse config from", *confFile, "failed, err is", err)
		return -1
	}

	d := new(daemon.Context)
	var c *os.Process
	if app.GsConfig.Daemon {
		core.GsTrace.Println("run in daemon mode, log file", app.GsConfig.Log.File)
		if child, err := d.Reborn(); err != nil {
			core.GsError.Println("daemon failed. err is", err)
			return -1
		} else {
			c = child
		}
	}
	defer d.Release()

	if c != nil {
		os.Exit(0)
	}

	return serve(svr)
}

func main() {
	ret := run()
	os.Exit(ret)
}
