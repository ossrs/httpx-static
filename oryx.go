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
	"fmt"
	"os"

	ohttp "github.com/ossrs/go-oryx-lib/http"
	ooptions "github.com/ossrs/go-oryx-lib/options"
	"github.com/ossrs/go-oryx/agent"
	"github.com/ossrs/go-oryx/app"
	"github.com/ossrs/go-oryx/core"
)

func serve(svr *app.Server, ctx core.Context) int {
	if err := svr.PrepareLogger(); err != nil {
		core.Error.Println(ctx, "prepare logger failed, err is", err)
		return -1
	}

	oryxMain(svr, ctx)

	core.Trace.Println(ctx, core.OryxSigServer(), core.OryxSigCopyright)
	core.Trace.Println(ctx, core.OryxSigProduct)

	if err := svr.Initialize(); err != nil {
		core.Error.Println(ctx, "initialize server failed, err is", err)
		return -1
	}

	if err := svr.Run(); err != nil {
		core.Error.Println(ctx, "run server failed, err is", err)
		return -1
	}

	return 0
}

func main() {
	// initialize global varialbes.
	core.RewriteLogger()
	ohttp.Server = core.OryxSigServer()

	// create application objects.
	ctx := core.NewContext()
	core.Conf = core.NewConfig(ctx)
	agent.Manager = agent.NewManager(ctx)

	// parse options and config.
	confFile := ooptions.ParseArgv("oryx.json", core.Version(), core.OryxSigServer())
	fmt.Println(fmt.Sprintf("%v signature is %v", core.OryxSigName, core.OryxSigServer()))

	ret := func() int {
		svr := app.NewServer(ctx)
		defer svr.Close()

		if err := svr.ParseConfig(confFile); err != nil {
			core.Error.Println(ctx, "parse config from", confFile, "failed, err is", err)
			return -1
		}

		return run(svr, ctx)
	}()

	os.Exit(ret)
}
