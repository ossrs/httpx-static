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
    core.LoggerTrace.Println(fmt.Sprintf("GO-SRS/%v is a golang implementation of SRS.", core.Version))
    flag.Parse()

    core.LoggerInfo.Println("start to parse config file", confFile)
    if err := core.GsConfig.Loads(confFile); err != nil {
        core.LoggerError.Println("parse config", confFile, "failed, err is", err)
        return -1
    }

    // reload goroutine
    go core.GsConfig.ReloadWorker(confFile)

    core.LoggerTrace.Println("Copyright (c) 2013-2015 SRS(simple-rtmp-server)")
    return core.ServerRun(core.GsConfig, func() int {
        return 0
    })
}

func main() {
    ret := run()
    os.Exit(ret)
}
