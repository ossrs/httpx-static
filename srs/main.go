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
    "os"
    "flag"
    "github.com/simple-rtmp-server/go-srs/core"
    "fmt"
)

// the startup argv:
//      -c conf/srs.json
//      --c conf/srs.json
//      -c=conf/srs.json
//      --c=conf/srs.json
var confFile = *flag.String("c", "conf/srs.json", "the config file.")

func run() int {
    flag.Parse()

    svr := core.NewServer()
    defer svr.Close()

    if err := svr.ParseConfig(confFile); err != nil {
        core.LoggerError.Println("parse config from", confFile, "failed, err is", err)
        return -1
    }

    if err := svr.PrepareLogger(); err != nil {
        core.LoggerError.Println("prepare logger failed, err is", err)
        return -1
    }

    core.LoggerTrace.Println("Copyright (c) 2013-2015 SRS(simple-rtmp-server)")
    core.LoggerTrace.Println(fmt.Sprintf("GO-SRS/%v is a golang implementation of SRS.", core.Version))

    if err := svr.Initialize(); err != nil {
        core.LoggerError.Println("initialize server failed, err is", err)
        return -1
    }

    if err := svr.Run(); err != nil {
        core.LoggerError.Println("run server failed, err is", err)
        return -1
    }

    return 0
}

func main() {
    ret := run()
    os.Exit(ret)
}
