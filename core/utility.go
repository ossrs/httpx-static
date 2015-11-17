// The MIT License (MIT)
//
// Copyright (c) 2013-2015 Oryx(ossrs)
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
	"math/rand"
	"time"
)

// the random object to fill bytes.
var random *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// randome fill the bytes.
func RandomFill(b []byte) {
	for i := 0; i < len(b); i++ {
		// the common value in [0x0f, 0xf0]
		b[i] = byte(0x0f + (random.Int() % (256 - 0x0f - 0x0f)))
	}
}

// invoke the f with recover.
// the name of goroutine, use empty to ignore.
func Recover(name string, f func() error) {
	defer func() {
		if r := recover(); r != nil {
			if name != "" {
				Warn.Println(name, "abort with", r)
			} else {
				Warn.Println("goroutine abort with", r)
			}
		}
	}()

	if err := f(); err != nil {
		if name != "" {
			Warn.Println(name, "terminated with", err)
		} else {
			Warn.Println("terminated abort with", err)
		}
	}
}
