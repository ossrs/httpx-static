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

package core

import (
	"io/ioutil"
	"log"
	"os"
)

const (
	logLabel      = "[gsrs]"
	LogInfoLabel  = logLabel + "[info] "
	LogTraceLabel = logLabel + "[trace] "
	LogWarnLabel  = logLabel + "[warn] "
	LogErrorLabel = logLabel + "[error] "
)

// the application loggers
// info, the verbose info level, very detail log, the lowest level, to discard.
var GsInfo Logger = log.New(ioutil.Discard, LogInfoLabel, log.LstdFlags)

// trace, the trace level, something important, the default log level, to stdout.
var GsTrace Logger = log.New(os.Stdout, LogTraceLabel, log.LstdFlags)

// warn, the warning level, dangerous information, to stderr.
var GsWarn Logger = log.New(os.Stderr, LogWarnLabel, log.LstdFlags)

// error, the error level, fatal error things, ot stderr.
var GsError Logger = log.New(os.Stderr, LogErrorLabel, log.LstdFlags)

// the logger for gsrs.
type Logger interface {
	Println(a ...interface{})
}
