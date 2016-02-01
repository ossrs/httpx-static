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

package app

import (
	"github.com/ossrs/go-oryx/core"
	"log"
	"os"
)

// the simple logger which implements the interface
// and log to console or file.
type simpleLogger struct {
	ctx  core.Context
	file *os.File
}

// TODO: FIXME: maybe we can got goroutine id by reflect.
func (v *simpleLogger) open(c *core.Config) (err error) {
	ctx := v.ctx

	core.Info.Println(ctx, "apply log tank", c.Log.Tank)
	core.Info.Println(ctx, "apply log level", c.Log.Level)

	if c.LogToFile() {
		core.Trace.Println(ctx, "apply log", c.Log.Tank, c.Log.Level, c.Log.File)
		core.Trace.Println(ctx, "please see detail of log: tailf", c.Log.File)

		if v.file, err = os.OpenFile(c.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
			core.Error.Println(ctx, "open log file", c.Log.File, "failed, err is", err)
			return
		} else {
			core.Info = core.NewLoggerPlus(log.New(c.LogTank("info", v.file), core.LogInfoLabel, log.LstdFlags))
			core.Trace = core.NewLoggerPlus(log.New(c.LogTank("trace", v.file), core.LogTraceLabel, log.LstdFlags))
			core.Warn = core.NewLoggerPlus(log.New(c.LogTank("warn", v.file), core.LogWarnLabel, log.LstdFlags))
			core.Error = core.NewLoggerPlus(log.New(c.LogTank("error", v.file), core.LogErrorLabel, log.LstdFlags))
		}
	} else {
		core.Trace.Println(ctx, "apply log", c.Log.Tank, c.Log.Level)

		core.Info = core.NewLoggerPlus(log.New(c.LogTank("info", os.Stdout), core.LogInfoLabel, log.LstdFlags))
		core.Trace = core.NewLoggerPlus(log.New(c.LogTank("trace", os.Stdout), core.LogTraceLabel, log.LstdFlags))
		core.Warn = core.NewLoggerPlus(log.New(c.LogTank("warn", os.Stderr), core.LogWarnLabel, log.LstdFlags))
		core.Error = core.NewLoggerPlus(log.New(c.LogTank("error", os.Stderr), core.LogErrorLabel, log.LstdFlags))
	}

	return
}

func (v *simpleLogger) close(c *core.Config) (err error) {
	ctx := v.ctx

	if v.file == nil {
		return
	}

	// when log closed, set the logger warn to stderr for file closed.
	core.Warn = core.NewLoggerPlus(log.New(os.Stderr, core.LogWarnLabel, log.LstdFlags))

	// try to close the log file.
	if err = v.file.Close(); err != nil {
		core.Warn.Println(ctx, "gracefully close log file", c.Log.File, "failed, err is", err)
	} else {
		core.Warn.Println(ctx, "close log file", c.Log.File, "ok")
	}

	return
}
