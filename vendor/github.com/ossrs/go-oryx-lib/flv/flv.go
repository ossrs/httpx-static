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

// The oryx FLV package support bytes from/to FLV tags.
package flv

import (
	"bytes"
	"errors"
	"github.com/ossrs/go-oryx-lib/aac"
	"io"
)

// FLV Tag Type is the type of tag,
// refer to @doc video_file_format_spec_v10.pdf, @page 9, @section FLV tags
type TagType uint8

const (
	TagTypeForbidden  TagType = 0
	TagTypeAudio      TagType = 8
	TagTypeVideo      TagType = 9
	TagTypeScriptData TagType = 18
)

func (v TagType) String() string {
	switch v {
	case TagTypeVideo:
		return "Video"
	case TagTypeAudio:
		return "Audio"
	case TagTypeScriptData:
		return "Data"
	default:
		return "Forbidden"
	}
}

// FLV Demuxer is used to demux FLV file.
// Refer to @doc video_file_format_spec_v10.pdf, @page 74, @section Annex E. The FLV File Format
// A FLV file must consist the bellow parts:
//	1. A FLV header, refer to @doc video_file_format_spec_v10.pdf, @page 8, @section The FLV header
//	2. One or more tags, refer to @doc video_file_format_spec_v10.pdf, @page 9, @section FLV tags
// @remark We always ignore the previous tag size.
type Demuxer interface {
	// Read the FLV header, return the version of FLV, whether hasVideo or hasAudio in header.
	ReadHeader() (version uint8, hasVideo, hasAudio bool, err error)
	// Read the FLV tag header, return the tag information, especially the tag size,
	// then user can read the tag payload.
	ReadTagHeader() (tagType TagType, tagSize, timestamp uint32, err error)
	// Read the FLV tag body, drop the next 4 bytes previous tag size.
	ReadTag(tagSize uint32) (tag []byte, err error)
	// Close the demuxer.
	Close() error
}

// When FLV signature is not "FLV"
var errSignature = errors.New("FLV signatures are illegal")

// Create a demuxer object.
func NewDemuxer(r io.Reader) (Demuxer, error) {
	return &demuxer{
		r: r,
	}, nil
}

type demuxer struct {
	r io.Reader
}

func (v *demuxer) ReadHeader() (version uint8, hasVideo, hasAudio bool, err error) {
	h := &bytes.Buffer{}
	if _, err = io.CopyN(h, v.r, 13); err != nil {
		return
	}

	p := h.Bytes()

	if !bytes.Equal([]byte{byte('F'), byte('L'), byte('V')}, p[:3]) {
		err = errSignature
		return
	}

	version = uint8(p[3])
	hasVideo = (p[4] & 0x01) == 0x01
	hasAudio = ((p[4] >> 2) & 0x01) == 0x01

	return
}

func (v *demuxer) ReadTagHeader() (tagType TagType, tagSize uint32, timestamp uint32, err error) {
	h := &bytes.Buffer{}
	if _, err = io.CopyN(h, v.r, 11); err != nil {
		return
	}

	p := h.Bytes()

	tagType = TagType(p[0])
	tagSize = uint32(p[1])<<16 | uint32(p[2])<<8 | uint32(p[3])
	timestamp = uint32(p[7])<<24 | uint32(p[4])<<16 | uint32(p[5])<<8 | uint32(p[6])

	return
}

func (v *demuxer) ReadTag(tagSize uint32) (tag []byte, err error) {
	h := &bytes.Buffer{}
	if _, err = io.CopyN(h, v.r, int64(tagSize+4)); err != nil {
		return
	}

	p := h.Bytes()
	tag = p[0 : len(p)-4]

	return
}

func (v *demuxer) Close() error {
	return nil
}

// The FLV muxer is used to write packet in FLV protocol.
// Refer to @doc video_file_format_spec_v10.pdf, @page 74, @section Annex E. The FLV File Format
type Muxer interface {
	// Write the FLV header.
	WriteHeader(hasVideo, hasAudio bool) (err error)
	// Write A FLV tag.
	WriteTag(tagType TagType, timestamp uint32, tag []byte) (err error)
	// Close the muxer.
	Close() error
}

// Create a muxer object.
func NewMuxer(w io.Writer) (Muxer, error) {
	return &muxer{
		w: w,
	}, nil
}

type muxer struct {
	w io.Writer
}

func (v *muxer) WriteHeader(hasVideo, hasAudio bool) (err error) {
	var flags byte
	if hasVideo {
		flags |= 0x01
	}
	if hasAudio {
		flags |= 0x04
	}

	r := bytes.NewReader([]byte{
		byte('F'), byte('L'), byte('V'),
		0x01,
		flags,
		0x00, 0x00, 0x00, 0x09,
		0x00, 0x00, 0x00, 0x00,
	})

	if _, err = io.Copy(v.w, r); err != nil {
		return
	}

	return
}

func (v *muxer) WriteTag(tagType TagType, timestamp uint32, tag []byte) (err error) {
	// Tag header.
	tagSize := uint32(len(tag))

	r := bytes.NewReader([]byte{
		byte(tagType),
		byte(tagSize >> 16), byte(tagSize >> 8), byte(tagSize),
		byte(timestamp >> 16), byte(timestamp >> 8), byte(timestamp),
		byte(timestamp >> 24),
		0x00, 0x00, 0x00,
	})

	if _, err = io.Copy(v.w, r); err != nil {
		return
	}

	// TAG
	if _, err = io.Copy(v.w, bytes.NewReader(tag)); err != nil {
		return
	}

	// Previous tag size.
	pts := uint32(11 + len(tag))
	r = bytes.NewReader([]byte{
		byte(pts >> 24), byte(pts >> 16), byte(pts >> 8), byte(pts),
	})

	if _, err = io.Copy(v.w, r); err != nil {
		return
	}

	return
}

func (v *muxer) Close() error {
	return nil
}

// The Audio AAC frame trait, whether sequence header(ASC) or raw data.
// Refer to @doc video_file_format_spec_v10.pdf, @page 77, @section E.4.2 Audio Tags
type AudioFrameTrait uint8

const (
	AudioFrameTraitSequenceHeader AudioFrameTrait = iota // 0 = AAC sequence header
	AudioFrameTraitRaw                                   // 1 = AAC raw
	AudioFrameTraitForbidden
)

func (v AudioFrameTrait) String() string {
	switch v {
	case AudioFrameTraitSequenceHeader:
		return "SequenceHeader"
	case AudioFrameTraitRaw:
		return "Raw"
	default:
		return "Forbidden"
	}
}

// The audio channels, FLV named it the SoundType.
// Refer to @doc video_file_format_spec_v10.pdf, @page 77, @section E.4.2 Audio Tags
type AudioChannels uint8

const (
	AudioChannelsMono   AudioChannels = iota // 0 = Mono sound
	AudioChannelsStereo                      // 1 = Stereo sound
	AudioChannelsForbidden
)

func (v AudioChannels) String() string {
	switch v {
	case AudioChannelsMono:
		return "Mono"
	case AudioChannelsStereo:
		return "Stereo"
	default:
		return "Forbidden"
	}
}

func (v *AudioChannels) From(a aac.Channels) {
	switch a {
	case aac.ChannelMono:
		*v = AudioChannelsMono
	case aac.ChannelStereo:
		*v = AudioChannelsStereo
	case aac.Channel3, aac.Channel4, aac.Channel5, aac.Channel5_1, aac.Channel7_1:
		*v = AudioChannelsStereo
	default:
		*v = AudioChannelsForbidden
	}
}

// The audio sample bits, FLV named it the SoundSize.
// Refer to @doc video_file_format_spec_v10.pdf, @page 76, @section E.4.2 Audio Tags
type AudioSampleBits uint8

const (
	AudioSampleBits8bits  AudioSampleBits = iota // 0 = 8-bit samples
	AudioSampleBits16bits                        // 1 = 16-bit samples
	AudioSampleBitsForbidden
)

func (v AudioSampleBits) String() string {
	switch v {
	case AudioSampleBits8bits:
		return "8-bits"
	case AudioSampleBits16bits:
		return "16-bits"
	default:
		return "Forbidden"
	}
}

// The audio sampling rate, FLV named it the SoundRate.
// Refer to @doc video_file_format_spec_v10.pdf, @page 76, @section E.4.2 Audio Tags
type AudioSamplingRate uint8

const (
	AudioSamplingRate5kHz  AudioSamplingRate = iota // 0 = 5.5 kHz
	AudioSamplingRate11kHz                          // 1 = 11 kHz
	AudioSamplingRate22kHz                          // 2 = 22 kHz
	AudioSamplingRate44kHz                          // 3 = 44 kHz
	AudioSamplingRateForbidden
)

func (v AudioSamplingRate) String() string {
	switch v {
	case AudioSamplingRate5kHz:
		return "5.5kHz"
	case AudioSamplingRate11kHz:
		return "11kHz"
	case AudioSamplingRate22kHz:
		return "22kHz"
	case AudioSamplingRate44kHz:
		return "44kHz"
	default:
		return "Forbidden"
	}
}

// Parse the FLV sampling rate to Hz.
func (v AudioSamplingRate) ToHz() int {
	flvSR := []int{5512, 11025, 22050, 44100}
	return flvSR[v]
}

// Convert aac sample rate index to FLV sampling rate.
func (v *AudioSamplingRate) From(a aac.SampleRateIndex) {
	switch a {
	case aac.SampleRateIndex96kHz, aac.SampleRateIndex88kHz, aac.SampleRateIndex64kHz:
		*v = AudioSamplingRate44kHz
	case aac.SampleRateIndex48kHz, aac.SampleRateIndex44kHz, aac.SampleRateIndex32kHz:
		*v = AudioSamplingRate44kHz
	case aac.SampleRateIndex24kHz, aac.SampleRateIndex22kHz, aac.SampleRateIndex16kHz:
		*v = AudioSamplingRate22kHz
	case aac.SampleRateIndex12kHz, aac.SampleRateIndex11kHz, aac.SampleRateIndex8kHz:
		*v = AudioSamplingRate11kHz
	case aac.SampleRateIndex7kHz:
		*v = AudioSamplingRate5kHz
	default:
		*v = AudioSamplingRateForbidden
	}
}

// The audio codec id, FLV named it the SoundFormat.
// Refer to @doc video_file_format_spec_v10.pdf, @page 76, @section E.4.2 Audio Tags
type AudioCodec uint8

const (
	AudioCodecLinearPCM       AudioCodec = iota // 0 = Linear PCM, platform endian
	AudioCodecADPCM                             // 1 = ADPCM
	AudioCodecMP3                               // 2 = MP3
	AudioCodecLinearPCMle                       // 3 = Linear PCM, little endian
	AudioCodecNellymoser16kHz                   // 4 = Nellymoser 16 kHz mono
	AudioCodecNellymoser8kHz                    // 5 = Nellymoser 8 kHz mono
	AudioCodecNellymoser                        // 6 = Nellymoser
	AudioCodecG711Alaw                          // 7 = G.711 A-law logarithmic PCM
	AudioCodecG711MuLaw                         // 8 = G.711 mu-law logarithmic PCM
	AudioCodecReserved                          // 9 = reserved
	AudioCodecAAC                               // 10 = AAC
	AudioCodecSpeex                             // 11 = Speex
	AudioCodecUndefined12
	AudioCodecUndefined13
	AudioCodecMP3In8kHz      // 14 = MP3 8 kHz
	AudioCodecDeviceSpecific // 15 = Device-specific sound
	AudioCodecForbidden
)

func (v AudioCodec) String() string {
	switch v {
	case AudioCodecLinearPCM:
		return "LinearPCM(platform-endian)"
	case AudioCodecADPCM:
		return "ADPCM"
	case AudioCodecMP3:
		return "MP3"
	case AudioCodecLinearPCMle:
		return "LinearPCM(little-endian)"
	case AudioCodecNellymoser16kHz:
		return "Nellymoser(16kHz-mono)"
	case AudioCodecNellymoser8kHz:
		return "Nellymoser(8kHz-mono)"
	case AudioCodecNellymoser:
		return "Nellymoser"
	case AudioCodecG711Alaw:
		return "G.711(A-law)"
	case AudioCodecG711MuLaw:
		return "G.711(mu-law)"
	case AudioCodecAAC:
		return "AAC"
	case AudioCodecSpeex:
		return "Speex"
	case AudioCodecMP3In8kHz:
		return "MP3(8kHz)"
	case AudioCodecDeviceSpecific:
		return "DeviceSpecific"
	default:
		return "Forbidden"
	}
}

type AudioFrame struct {
	SoundFormat AudioCodec
	SoundRate   AudioSamplingRate
	SoundSize   AudioSampleBits
	SoundType   AudioChannels
	Trait       AudioFrameTrait
	Raw         []byte
}

// The packager used to codec the FLV audio tag body.
// Refer to @doc video_file_format_spec_v10.pdf, @page 76, @section E.4.2 Audio Tags
type AudioPackager interface {
	// Encode the audio frame to FLV audio tag.
	Encode(frame *AudioFrame) (tag []byte, err error)
	// Decode the FLV audio tag to audio frame.
	Decode(tag []byte) (frame *AudioFrame, err error)
}

var errDataNotEnough = errors.New("Data not enough")

type audioPackager struct {
}

func NewAudioPackager() (AudioPackager, error) {
	return &audioPackager{}, nil
}

func (v *audioPackager) Encode(frame *AudioFrame) (tag []byte, err error) {
	if frame.SoundFormat == AudioCodecAAC {
		return append([]byte{
			byte(frame.SoundFormat)<<4 | byte(frame.SoundRate)<<2 | byte(frame.SoundSize)<<1 | byte(frame.SoundType),
			byte(frame.Trait),
		}, frame.Raw...), nil
	} else {
		return append([]byte{
			byte(frame.SoundFormat)<<4 | byte(frame.SoundRate)<<2 | byte(frame.SoundSize)<<1 | byte(frame.SoundType),
		}, frame.Raw...), nil
	}
}

func (v *audioPackager) Decode(tag []byte) (frame *AudioFrame, err error) {
	// Refer to @doc video_file_format_spec_v10.pdf, @page 76, @section E.4.2 Audio Tags
	// @see SrsFormat::audio_aac_demux
	if len(tag) < 2 {
		err = errDataNotEnough
		return
	}

	t := uint8(tag[0])
	frame = &AudioFrame{}
	frame.SoundFormat = AudioCodec(uint8(t>>4) & 0x0f)
	frame.SoundRate = AudioSamplingRate(uint8(t>>2) & 0x03)
	frame.SoundSize = AudioSampleBits(uint8(t>>1) & 0x01)
	frame.SoundType = AudioChannels(t & 0x01)

	if frame.SoundFormat == AudioCodecAAC {
		frame.Trait = AudioFrameTrait(tag[1])
		frame.Raw = tag[2:]
	} else {
		frame.Raw = tag[1:]
	}

	return
}

// The video frame type.
// Refer to @doc video_file_format_spec_v10.pdf, @page 78, @section E.4.3 Video Tags
type VideoFrameType uint8

const (
	VideoFrameTypeForbidden  VideoFrameType = iota
	VideoFrameTypeKeyframe                  //  1 = key frame (for AVC, a seekable frame)
	VideoFrameTypeInterframe                // 2 = inter frame (for AVC, a non-seekable frame)
	VideoFrameTypeDisposable                // 3 = disposable inter frame (H.263 only)
	VideoFrameTypeGenerated                 // 4 = generated key frame (reserved for server use only)
	VideoFrameTypeInfo                      // 5 = video info/command frame
)

func (v VideoFrameType) String() string {
	switch v {
	case VideoFrameTypeKeyframe:
		return "Keyframe"
	case VideoFrameTypeInterframe:
		return "Interframe"
	case VideoFrameTypeDisposable:
		return "DisposableInterframe"
	case VideoFrameTypeGenerated:
		return "GeneratedKeyframe"
	case VideoFrameTypeInfo:
		return "Info"
	default:
		return "Forbidden"
	}
}

// The video codec id.
// Refer to @doc video_file_format_spec_v10.pdf, @page 78, @section E.4.3 Video Tags
type VideoCodec uint8

const (
	VideoCodecForbidden   VideoCodec = iota + 1
	VideoCodecH263                   // 2 = Sorenson H.263
	VideoCodecScreen                 // 3 = Screen video
	VideoCodecOn2VP6                 // 4 = On2 VP6
	VideoCodecOn2VP6Alpha            // 5 = On2 VP6 with alpha channel
	VideoCodecScreen2                // 6 = Screen video version 2
	VideoCodecAVC                    // 7 = AVC
)

func (v VideoCodec) String() string {
	switch v {
	case VideoCodecH263:
		return "H.263"
	case VideoCodecScreen:
		return "Screen"
	case VideoCodecOn2VP6:
		return "VP6"
	case VideoCodecOn2VP6Alpha:
		return "On2VP6(alpha)"
	case VideoCodecScreen2:
		return "Screen2"
	case VideoCodecAVC:
		return "AVC"
	default:
		return "Forbidden"
	}
}

// The video AVC frame trait, whethere sequence header or not.
// Refer to @doc video_file_format_spec_v10.pdf, @page 78, @section E.4.3 Video Tags
type VideoFrameTrait uint8

const (
	VideoFrameTraitSequenceHeader VideoFrameTrait = iota // 0 = AVC sequence header
	VideoFrameTraitNALU                                  // 1 = AVC NALU
	VideoFrameTraitSequenceEOF                           // 2 = AVC end of sequence (lower level NALU sequence ender is
	VideoFrameTraitForbidden
)

func (v VideoFrameTrait) String() string {
	switch v {
	case VideoFrameTraitSequenceHeader:
		return "SequenceHeader"
	case VideoFrameTraitNALU:
		return "NALU"
	case VideoFrameTraitSequenceEOF:
		return "SequenceEOF"
	default:
		return "Forbidden"
	}
}

type VideoFrame struct {
	CodecID   VideoCodec
	FrameType VideoFrameType
	Trait     VideoFrameTrait
	CTS       int32
	Raw       []byte
}

// The packager used to codec the FLV video tag body.
// Refer to @doc video_file_format_spec_v10.pdf, @page 78, @section E.4.3 Video Tags
type VideoPackager interface {
	// Decode the FLV video tag to video frame.
	// @remark For RTMP/FLV: pts = dts + cts, where dts is timestamp in packet/tag.
	Decode(tag []byte) (frame *VideoFrame, err error)
	// Encode the video frame to FLV video tag.
	Encode(frame *VideoFrame) (tag []byte, err error)
}

type videoPackager struct {
}

func NewVideoPackager() (VideoPackager, error) {
	return &videoPackager{}, nil
}

func (v *videoPackager) Decode(tag []byte) (frame *VideoFrame, err error) {
	if len(tag) < 5 {
		err = errDataNotEnough
		return
	}

	p := tag
	frame = &VideoFrame{}
	frame.FrameType = VideoFrameType(byte(p[0]>>4) & 0x0f)
	frame.CodecID = VideoCodec(byte(p[0]) & 0x0f)

	if frame.CodecID == VideoCodecAVC {
		frame.Trait = VideoFrameTrait(p[1])
		frame.CTS = int32(uint32(p[2])<<16 | uint32(p[3])<<8 | uint32(p[4]))
		frame.Raw = tag[5:]
	} else {
		frame.Raw = tag[1:]
	}

	return
}

func (v videoPackager) Encode(frame *VideoFrame) (tag []byte, err error) {
	if frame.CodecID == VideoCodecAVC {
		return append([]byte{
			byte(frame.FrameType)<<4 | byte(frame.CodecID), byte(frame.Trait),
			byte(frame.CTS >> 16), byte(frame.CTS >> 8), byte(frame.CTS),
		}, frame.Raw...), nil
	} else {
		return append([]byte{
			byte(frame.FrameType)<<4 | byte(frame.CodecID),
		}, frame.Raw...), nil
	}
}
