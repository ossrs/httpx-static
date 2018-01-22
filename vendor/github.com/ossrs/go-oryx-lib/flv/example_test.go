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

package flv_test

import (
	"github.com/ossrs/go-oryx-lib/flv"
	"io"
)

func ExampleDemuxer() {
	// To open a flv file, or http flv stream.
	var r io.Reader

	var err error
	var f flv.Demuxer
	if f, err = flv.NewDemuxer(r); err != nil {
		return
	}
	defer f.Close()

	var version uint8
	var hasVideo, hasAudio bool
	if version, hasVideo, hasAudio, err = f.ReadHeader(); err != nil {
		return
	}

	// Optional, user can check the header.
	_ = version
	_ = hasAudio
	_ = hasVideo

	var tagType flv.TagType
	var tagSize, timestamp uint32
	if tagType, tagSize, timestamp, err = f.ReadTagHeader(); err != nil {
		return
	}

	var tag []byte
	if tag, err = f.ReadTag(tagSize); err != nil {
		return
	}

	// Using the FLV tag type, dts and body.
	// Refer to @doc video_file_format_spec_v10.pdf, @page 9, @section FLV tags
	_ = tagType
	_ = timestamp
	_ = tag
}

func ExampleMuxer() {
	// To open a flv file or http post stream.
	var w io.Writer

	var err error
	var f flv.Muxer
	if f, err = flv.NewMuxer(w); err != nil {
		return
	}
	defer f.Close()

	if err = f.WriteHeader(true, true); err != nil {
		return
	}

	var tagType flv.TagType
	var timestamp uint32
	var tag []byte
	// Get a FLV tag to write to muxer.
	if err = f.WriteTag(tagType, timestamp, tag); err != nil {
		return
	}
}
