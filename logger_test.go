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
    "io/ioutil"
)

func TestBasicLogger(t *testing.T) {
    GsInfo = log.New(ioutil.Discard, logLabel, log.LstdFlags)
    GsTrace = log.New(ioutil.Discard, logLabel, log.LstdFlags)
    GsWarn = log.New(ioutil.Discard, logLabel, log.LstdFlags)
    GsError = log.New(ioutil.Discard, logLabel, log.LstdFlags)

    GsInfo.Println("test logger info ok.")
    GsTrace.Println("test logger trace ok.")
    GsWarn.Println("test logger warn ok.")
    GsError.Println("test logger error ok.")
}
