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
 This the main entrance of flvlb, load-balance for flv streaming.
*/
package main

import (
	"encoding/json"
	"fmt"
	oj "github.com/ossrs/go-oryx-lib/json"
	ol "github.com/ossrs/go-oryx-lib/logger"
	oo "github.com/ossrs/go-oryx-lib/options"
	"github.com/ossrs/go-oryx/kernel"
	"os"
)

var signature = fmt.Sprintf("FLVLB/%v", kernel.Version())

// The config object for flvlb module.
type FlvLbConfig struct {
	kernel.Config
}

func (v *FlvLbConfig) String() string {
	return fmt.Sprintf("%v", &v.Config)
}

func (v *FlvLbConfig) Loads(c string) (err error) {
	var f *os.File
	if f, err = os.Open(c); err != nil {
		fmt.Println("Open config failed, err is", err)
		return
	}
	defer f.Close()

	r := json.NewDecoder(oj.NewJsonPlusReader(f))
	if err = r.Decode(v); err != nil {
		fmt.Println("Decode config failed, err is", err)
		return
	}

	if err = v.Config.OpenLogger(); err != nil {
		fmt.Println("Open logger failed, err is", err)
		return
	}

	return
}

func main() {
	var err error
	confFile := oo.ParseArgv("conf/flv.lb.json", kernel.Version(), signature)
	fmt.Println("FLVLB is the load-balance for flv streaming, config is", confFile)

	conf := &FlvLbConfig{}
	if err = conf.Loads(confFile); err != nil {
		fmt.Println("Loads config failed, err is", err)
		return
	}
	defer conf.Close()

	ctx := &kernel.Context{}
	ol.T(ctx, fmt.Sprintf("Config ok, %v", conf))

	return
}
