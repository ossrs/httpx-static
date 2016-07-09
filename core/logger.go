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

package core

import (
	ocore "github.com/ossrs/go-oryx-lib/logger"
	"io/ioutil"
	"log"
	"os"
)

// Context alias of the Context interface.
// @remark user can directly use ocore Context.
type Context interface {
	ocore.Context
}

// implements the context.
// @remark user can use nil context.
type context int

var _Cid = 100

// NewContext provides a context with current Cid, incrementing Cid, and returning the context
func NewContext() Context {
	v := context(_Cid)
	_Cid++
	return v
}

func (v context) Cid() int {
	return int(v)
}

// Logger alias the Logger interface.
// @remark user can directly use ocore Logger.
type Logger interface {
	ocore.Logger
}

// NewLoggerPlus alias for log plus.
func NewLoggerPlus(l *log.Logger) Logger {
	return Logger(ocore.NewLoggerPlus(l))
}

// the application loggers

// Info the verbose info level, very detail log, the lowest level, to discard.
var Info Logger

// Trace the trace level, something important, the default log level, to stdout.
var Trace Logger

// Warn the warning level, dangerous information, to stderr.
var Warn Logger

// Error the error level, fatal error things, ot stderr.
var Error Logger

const (
	logLabel = "[oryx]"
	// LogInfoLabel provides Info labeling for logs
	LogInfoLabel = logLabel + "[info] "
	// LogTraceLabel provides Trace labeling for logs
	LogTraceLabel = logLabel + "[trace] "
	// LogWarnLabel provides Warn labeling for logs
	LogWarnLabel = logLabel + "[warn] "
	// LogErrorLabel provides Error labeling for logs
	LogErrorLabel = logLabel + "[error] "
)

// RewriteLogger rewrites the label and sets an alias for the logger.
// @remark for normal application, use the ocore directly.
func RewriteLogger() {
	// rewrite the label for ocore.
	ocore.Info = ocore.NewLoggerPlus(log.New(ioutil.Discard, LogInfoLabel, log.LstdFlags))
	ocore.Trace = ocore.NewLoggerPlus(log.New(os.Stdout, LogTraceLabel, log.LstdFlags))
	ocore.Warn = ocore.NewLoggerPlus(log.New(os.Stderr, LogWarnLabel, log.LstdFlags))
	ocore.Error = ocore.NewLoggerPlus(log.New(os.Stderr, LogErrorLabel, log.LstdFlags))

	// alias core logger to ocore.
	Info = ocore.Info
	Trace = ocore.Trace
	Warn = ocore.Warn
	Error = ocore.Error
}
