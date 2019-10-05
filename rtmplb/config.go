/*
The MIT License (MIT)

Copyright (c) 2016 winlin

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

/*
 This is the basic config for oryx.
*/
package main

import (
	"fmt"
	ol "github.com/ossrs/go-oryx-lib/logger"
	"os"
)

// The basic config, for all modules which will provides these config.
type Config struct {
	Logger struct {
		Tank     string `json:"tank"`
		FilePath string `json:"file"`
	} `json:"logger"`
}

// The interface fmt.Stringer
func (v *Config) String() string {
	var logger string
	if v.Logger.Tank == "console" {
		logger = v.Logger.Tank
	} else {
		logger = fmt.Sprintf("tank=%v,file=%v", v.Logger.Tank, v.Logger.FilePath)
	}

	return fmt.Sprintf("logger(tank=%v)", logger)
}

// The interface io.Closer
// Cleanup the resource open by config, for example, the logger file.
func (v *Config) Close() error {
	return ol.Close()
}

// Open the logger, when tank is file, switch logger to file.
func (v *Config) OpenLogger() (err error) {
	if tank := v.Logger.Tank; tank != "file" && tank != "console" {
		return fmt.Errorf("Invalid logger tank, must be console/file, actual is %v", tank)
	}

	if v.Logger.Tank != "file" {
		return
	}

	var f *os.File
	if f, err = os.OpenFile(v.Logger.FilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
		return fmt.Errorf("Open logger %v failed, err is", v.Logger.FilePath, err)
	}

	_ = ol.Close()
	ol.Switch(f)

	return
}
