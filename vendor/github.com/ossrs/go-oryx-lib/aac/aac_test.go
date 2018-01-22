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

package aac

import (
	"bytes"
	"testing"
)

func TestAudioSpecificConfig_MarshalBinary(t *testing.T) {
	asc := &AudioSpecificConfig{}

	if _, err := asc.MarshalBinary(); err == nil {
		t.Error("marshal")
	}

	asc.Object = ObjectTypeMain
	if _, err := asc.MarshalBinary(); err == nil {
		t.Error("marshal")
	}

	asc.SampleRate = SampleRateIndex44kHz
	if _, err := asc.MarshalBinary(); err == nil {
		t.Error("marshal")
	}

	asc.Channels = ChannelStereo
	if _, err := asc.MarshalBinary(); err != nil {
		t.Errorf("marshal failed %+v", err)
	}
}

func TestAudioSpecificConfig_UnmarshalBinary(t *testing.T) {
	asc := &AudioSpecificConfig{}

	if err := asc.UnmarshalBinary(nil); err == nil {
		t.Error("unmarshal")
	}

	if err := asc.UnmarshalBinary([]byte{0x12}); err == nil {
		t.Error("unmarshal")
	}

	if err := asc.UnmarshalBinary([]byte{0x12, 0x10}); err != nil {
		t.Errorf("%+v", err)
	}
}

func TestAdts_Encode(t *testing.T) {
	adts, err := NewADTS()
	if err != nil {
		t.Errorf("%+v", err)
	}

	if err = adts.SetASC([]byte{0x12, 0x10}); err != nil {
		t.Errorf("%+v", err)
	}

	if data, err := adts.Encode(nil); err != nil {
		t.Errorf("%+v", err)
	} else if len(data) != 7 {
		t.Error("encode")
	}

	if data, err := adts.Encode([]byte{0x00}); err != nil {
		t.Errorf("%+v", err)
	} else if len(data) != 8 {
		t.Error("encode")
	}
}

func TestAdts_Decode(t *testing.T) {
	adts, err := NewADTS()
	if err != nil {
		t.Errorf("%+v", err)
	}

	if raw, left, err := adts.Decode([]byte{
		0xff, 0xf1, 0x50, 0x80, 0x01, 0x00, 0xfc, 0x00,
	}); err != nil {
		t.Errorf("%+v", err)
	} else if bytes.Compare(raw, []byte{0x00}) != 0 {
		t.Errorf("%#x", raw)
	} else if len(left) != 0 {
		t.Errorf("%#x", left)
	}

	asc := adts.ASC()
	if asc.Object != ObjectTypeLC {
		t.Error(asc.Object)
	}
	if asc.SampleRate != SampleRateIndex44kHz {
		t.Error(asc.SampleRate)
	}
	if asc.Channels != ChannelStereo {
		t.Error(asc.Channels)
	}
}

func TestAdts_Decode2(t *testing.T) {
	adts, err := NewADTS()
	if err != nil {
		t.Errorf("%+v", err)
	}

	if raw, left, err := adts.Decode([]byte{
		0xff, 0xf1, 0x50, 0x80, 0x01, 0x00, 0xfc, 0x00, 0x01,
	}); err != nil {
		t.Errorf("%+v", err)
	} else if bytes.Compare(raw, []byte{0x00}) != 0 {
		t.Errorf("%#x", raw)
	} else if bytes.Compare(left, []byte{0x01}) != 0 {
		t.Errorf("%#x", left)
	}
}

func TestAdts_Decode3(t *testing.T) {
	adts, err := NewADTS()
	if err != nil {
		t.Errorf("%+v", err)
	}

	if _, _, err = adts.Decode(nil); err == nil {
		t.Error("decode")
	}

	if _, _, err = adts.Decode([]byte{0x00}); err == nil {
		t.Error("decode")
	}

	if _, _, err = adts.Decode(make([]byte, 7)); err == nil {
		t.Error("decode")
	}
}

func TestAdts_Decode4(t *testing.T) {
	adts, err := NewADTS()
	if err != nil {
		t.Errorf("%+v", err)
	}

	if _, _, err := adts.Decode([]byte{
		0xff, 0xf1, 0xff, 0x80, 0x01, 0x00, 0xfc, 0x00,
	}); err == nil {
		t.Error("decode")
	}
}
