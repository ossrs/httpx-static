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

package core

import (
    "runtime"
    "log"
    "os"
)

// commonly run helper.
func ServerRun(c *Config, callback func() int) int {
    LoggerTrace.Println("apply log tank", c.Log.Tank)
    LoggerTrace.Println("apply log level", c.Log.Level)
    if c.LogToFile() {
        LoggerTrace.Println("apply log file", c.Log.File)
        if f,err := os.OpenFile(c.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
            LoggerError.Println("open log file", c.Log.File, "failed, err is", err)
            return -1
        } else {
            defer func(){
                err = f.Close()

                // when log closed, set the logger warn to stderr for file closed.
                LoggerWarn = log.New(os.Stderr, "[gsrs][warn]", log.LstdFlags)

                if err != nil {
                    LoggerWarn.Println("gracefully close log file", c.Log.File, "failed, err is", err)
                } else {
                    LoggerWarn.Println("close log file", c.Log.File, "ok")
                }
            }()

            LoggerInfo = log.New(c.LogTank("info", f), logInfoLabel, log.LstdFlags)
            LoggerTrace = log.New(c.LogTank("trace", f), logTraceLabel, log.LstdFlags)
            LoggerWarn = log.New(c.LogTank("warn", f), logWarnLabel, log.LstdFlags)
            LoggerError = log.New(c.LogTank("error", f), logErrorLabel, log.LstdFlags)
        }
        LoggerTrace.Println("please see detail of log: tailf", c.Log.File)
    } else {
        LoggerInfo = log.New(c.LogTank("info", os.Stdout), logInfoLabel, log.LstdFlags)
        LoggerTrace = log.New(c.LogTank("trace", os.Stdout), logTraceLabel, log.LstdFlags)
        LoggerWarn = log.New(c.LogTank("warn", os.Stderr), logWarnLabel, log.LstdFlags)
        LoggerError = log.New(c.LogTank("error", os.Stderr), logErrorLabel, log.LstdFlags)
    }

    LoggerTrace.Println("apply workers", c.Workers)
    runtime.GOMAXPROCS(c.Workers)

    return callback()
}
