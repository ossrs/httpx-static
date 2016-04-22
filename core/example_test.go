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

package core_test

import (
	"bytes"
	"fmt"
	"time"

	"github.com/ossrs/go-oryx/core"
)

func ExampleConfig_Loads() {
	ctx := core.NewContext()
	c := core.NewConfig(ctx)
	c.SetDefaults()

	//if err := c.Loads("config.json"); err != nil {
	//    panic(err)
	//}

	fmt.Println("listen at", c.Listen)
	fmt.Println("workers is", c.Workers)
	if c.Go.GcInterval == 0 {
		fmt.Println("go gc use default interval.")
	}

	// Output:
	// listen at 1935
	// workers is 0
	// go gc use default interval.
}

// the goroutine cycle ignore any error.
func ExampleWorkerContainer_recoverable() {
	var wc core.WorkerContainer
	wc.GFork("myservice", func(wc core.WorkerContainer) {
		for {
			select {
			case <-time.After(3 * time.Second):
				// select other channel, do something cycle to get error.
				if err := error(nil); err != nil {
					// recoverable error, log it only and continue or return.
					continue
				}
			case <-wc.QC():
				// when got a quit signal, break the loop.
				// and must notify the container again for other workers
				// in container to quit.
				wc.Quit()
				return
			}
		}
	})
}

// the goroutine cycle absolutely safe, no panic no error to quit.
func ExampleWorkerContainer_safe() {
	var wc core.WorkerContainer
	wc.GFork("myservice", func(wc core.WorkerContainer) {
		defer func() {
			if r := recover(); r != nil {
				// log the r and ignore.
				return
			}
		}()

		for {
			select {
			case <-time.After(3 * time.Second):
				// select other channel, do something cycle to get error.
				if err := error(nil); err != nil {
					// recoverable error, log it only and continue or return.
					continue
				}
			case <-wc.QC():
				// when got a quit signal, break the loop.
				// and must notify the container again for other workers
				// in container to quit.
				wc.Quit()
				return
			}
		}
	})
}

// the goroutine cycle notify container to quit when error.
func ExampleWorkerContainer_fatal() {
	var wc core.WorkerContainer
	wc.GFork("myservice", func(wc core.WorkerContainer) {
		for {
			select {
			case <-time.After(3 * time.Second):
				// select other channel, do something cycle to get error.
				if err := error(nil); err != nil {
					// when got none-recoverable error, notify container to quit.
					wc.Quit()
					return
				}
			case <-wc.QC():
				// when got a quit signal, break the loop.
				// and must notify the container again for other workers
				// in container to quit.
				wc.Quit()
				return
			}
		}
	})
}

// marshal multiple objects to buffer.
func ExampleMarshal() {
	// objects to marshal
	var x core.Marshaler // for example NewAmf0String("oryx")
	var y core.Marshaler // for example NewAmf0Number(1.0)

	var b bytes.Buffer // marshal objects to b

	if err := core.Marshal(x, &b); err != nil {
		_ = err // when error.
	}
	if err := core.Marshal(y, &b); err != nil {
		_ = err // when error.
	}

	_ = b.Bytes() // use the bytes contains x and y
}

// unmarshal multiple objects from buffer
func ExampleUnmarshal() {
	var b bytes.Buffer // read from network.

	var x core.UnmarshalSizer // for example Amf0String
	var y core.UnmarshalSizer // for example Amf0Number

	if err := core.Unmarshal(x, &b); err != nil {
		_ = err // when error.
	}
	if err := core.Unmarshal(y, &b); err != nil {
		_ = err // when error.
	}

	// use x and y.
	_ = x
	_ = y
}
