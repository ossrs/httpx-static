// The MIT License (MIT)
//
// Copyright (c) 2013-2017 Oryx(ossrs)
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

package aac_test

import (
	"fmt"
	"github.com/ossrs/go-oryx-lib/aac"
)

func ExampleAdts_Decode() {
	var err error
	var adts aac.ADTS
	if adts, err = aac.NewADTS(); err != nil {
		fmt.Println(fmt.Sprintf("APP: Create ADTS failed, err is %+v", err))
		return
	}

	var data []byte // Read ADTS data from file or network.

	// Ignore the left, assume that the RAW only contains one AAC frame.
	var raw []byte
	if raw, _, err = adts.Decode(data); err != nil {
		fmt.Println(fmt.Sprintf("APP: ADTS decode failed, err is %+v", err))
		return
	}

	// Use the RAW data.
	_ = raw

	// Use the asc object, for example, used as RTMP audio sequence header.
	_ = adts.ASC()
}

func ExampleAdts_Encode() {
	var err error
	var adts aac.ADTS
	if adts, err = aac.NewADTS(); err != nil {
		fmt.Println(fmt.Sprintf("APP: Create ADTS failed, err is %+v", err))
		return
	}

	var raw []byte // Read RAW AAC from file or network.
	var data []byte
	if data, err = adts.Encode(raw); err != nil {
		fmt.Println(fmt.Sprintf("APP: ADTS encode failed, err is %+v", err))
		return
	}

	// Use the ADTS data.
	_ = data
}
