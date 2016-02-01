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

package agent

import (
	"fmt"
	"github.com/ossrs/go-oryx/core"
)

// the time jitter algorithm:
// 1. full, to ensure stream start at zero, and ensure stream monotonically increasing.
// 2. zero, only ensure sttream start at zero, ignore timestamp jitter.
// 3. off, disable the time jitter algorithm, like atc.
type JitterAlgorithm uint8

const (
	Full JitterAlgorithm = iota + 1
	Zero
	Off
)

// time jitter detect and correct,
// to ensure the stream is monotonically.
type Jitter struct {
	ctx                        core.Context
	lastPacketTimestamp        int64
	lastPacketCorrectTimestamp int64
}

func NewJitter(ctx core.Context) *Jitter {
	return &Jitter{
		ctx:                        ctx,
		lastPacketTimestamp:        -1,
		lastPacketCorrectTimestamp: -1,
	}
}

const (
	maxJitterMs     = 250
	maxJitterMsNeg  = -250
	frameIntervalMs = 10
)

func (v *Jitter) Correct(ts uint64, ag JitterAlgorithm) uint64 {
	ctx := v.ctx

	// for performance issue
	if ag != Full {
		// all jitter correct features is disabled, ignore.
		if ag == Off {
			return ts
		}

		// start at zero, but donot ensure monotonically increasing.
		if ag == Zero {
			// for the first time, last_pkt_correct_time is -1.
			if v.lastPacketCorrectTimestamp == -1 {
				v.lastPacketCorrectTimestamp = int64(ts)
			}
			return ts - uint64(v.lastPacketCorrectTimestamp)
		}

		// other algorithm, ignore.
		return ts
	}

	/**
	* we use a very simple time jitter detect/correct algorithm:
	* 1. delta: ensure the delta is positive and valid,
	*     we set the delta to DEFAULT_FRAME_TIME_MS,
	*     if the delta of time is nagative or greater than CONST_MAX_JITTER_MS.
	* 2. last_pkt_time: specifies the original packet time,
	*     is used to detect next jitter.
	* 3. last_pkt_correct_time: simply add the positive delta,
	*     and enforce the time monotonically.
	 */
	time := int64(ts)
	delta := int64(time - v.lastPacketTimestamp)

	// calc the correct timestamp.
	if v.lastPacketCorrectTimestamp == -1 {
		delta = -1 * v.lastPacketCorrectTimestamp // set to 1+(-1*(-1)) is zero.
	} else {
		// if jitter detected, reset the delta.
		if delta < maxJitterMsNeg || delta > maxJitterMs {
			// use default 10ms to notice the problem of stream.
			// @see https://github.com/ossrs/srs/issues/425
			delta = frameIntervalMs

			core.Trace.Println(ctx, fmt.Sprintf("jitter, last=%v, pts=%v, diff=%v, lastok=%v, ok=%v, delta=%v",
				v.lastPacketTimestamp, time, time-v.lastPacketTimestamp, v.lastPacketCorrectTimestamp,
				v.lastPacketCorrectTimestamp+delta, delta))
		}
	}

	// correct message.
	if v.lastPacketCorrectTimestamp += delta; v.lastPacketCorrectTimestamp < 0 {
		v.lastPacketCorrectTimestamp = 0
	}

	// update packet timestamp for next round-trip.
	v.lastPacketTimestamp = time

	return uint64(v.lastPacketCorrectTimestamp)
}
