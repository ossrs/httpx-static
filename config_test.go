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
    "fmt"
)

func TestConfigBasic(t *testing.T) {
    c := NewConfig()

    if c.Workers != Workers {
        t.Error("workers failed.")
    }

    if c.Listen != RtmpListen {
        t.Error("listen failed.")
    }

    if c.Go.GcInterval != GcIntervalSeconds {
        t.Error("go gc interval failed.")
    }

    if c.Log.Tank != "file" {
        t.Error("log tank failed.")
    }

    if c.Log.Level != "trace" {
        t.Error("log level failed.")
    }

    if c.Log.File != "gsrs.log" {
        t.Error("log file failed.")
    }
}

func BenchmarkConfigBasic(b *testing.B) {
    pc := NewConfig()
    cc := NewConfig()
    if err := pc.Reload(cc); err != nil {
        b.Error("reload failed.")
    }
}

func ExampleConfig_Loads() {
    c := NewConfig()

    //if err := c.Loads("config.json"); err != nil {
    //    panic(err)
    //}

    fmt.Println("listen at", c.Listen)
    fmt.Println("workers is", c.Workers)
    fmt.Println("go gc every", c.Go.GcInterval, "seconds")

    // Output:
    // listen at 1935
    // workers is 1
    // go gc every 300 seconds
}