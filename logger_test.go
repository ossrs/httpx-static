/*
The MIT License (MIT)

Copyright (c) 2013-2015 SRS(simple-rtmp-server)

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package main

import (
    "testing"
    "log"
    "strings"
)

// convert a func to interface io.Writer
type WriterFunc func(p []byte) (n int, err error)

// for io.Writer
func (f WriterFunc) Write(p []byte) (n int, err error) {
    return f(p)
}

func TestBasicLogger(t *testing.T) {
    var tank string
    var writer = func(p []byte) (n int, err error) {
        tank = string(p)
        return len(tank), nil
    }

    GsInfo = log.New(WriterFunc(writer), logInfoLabel, log.LstdFlags)
    GsTrace = log.New(WriterFunc(writer), logTraceLabel, log.LstdFlags)
    GsWarn = log.New(WriterFunc(writer), logWarnLabel, log.LstdFlags)
    GsError = log.New(WriterFunc(writer), logErrorLabel, log.LstdFlags)

    GsInfo.Println("test logger.")
    if !strings.HasPrefix(tank, "[gsrs][info]") {
        t.Error("logger format failed.")
    }
    if !strings.HasSuffix(tank, "test logger.\n") {
        t.Error("logger format failed. tank is", tank)
    }

    GsTrace.Println("test logger.")
    if !strings.HasPrefix(tank, "[gsrs][trace]") {
        t.Error("logger format failed.")
    }
    if !strings.HasSuffix(tank, "test logger.\n") {
        t.Error("logger format failed. tank is", tank)
    }

    GsWarn.Println("test logger.")
    if !strings.HasPrefix(tank, "[gsrs][warn]") {
        t.Error("logger format failed.")
    }
    if !strings.HasSuffix(tank, "test logger.\n") {
        t.Error("logger format failed. tank is", tank)
    }

    GsError.Println("test logger.")
    if !strings.HasPrefix(tank, "[gsrs][error]") {
        t.Error("logger format failed.")
    }
    if !strings.HasSuffix(tank, "test logger.\n") {
        t.Error("logger format failed. tank is", tank)
    }
}
