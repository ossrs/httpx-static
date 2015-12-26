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

import "fmt"

const (
	major     = 0
	minor     = 1
	reversion = 14
)

func Version() string {
	return fmt.Sprintf("%v.%v.%v", major, minor, reversion)
}

// stable major version
const OryxSigStable = 0

func OryxSigStableBranch() string {
	return fmt.Sprintf("%v.0release", OryxSigStable)
}

// project info.
const OryxSigKey = "SRS"
const OryxSigCode = "MonkeyKing"
const OryxSigRole = "cluster"
const OryxSigName = OryxSigKey + "(Simple RTMP Server)"
const OryxSigUrlShort = "github.com/ossrs/go-oryx"
const OryxSigUrl = "https://" + OryxSigUrlShort
const OryxSigWeb = "http://ossrs.net"
const OryxSigEmail = "winlin@vip.126.com"
const OryxSigLicense = "The MIT License (MIT)"
const OryxSigCopyright = "Copyright (c) 2013-2015 Oryx(ossrs)"
const OryxSigAuthors = "winlin"

func OryxSigPrimary() string {
	return fmt.Sprintf("Oryx/%v", OryxSigStable)
}
func OryxSigContributorsUrl() string {
	return fmt.Sprintf("%v/blob/%v/AUTHORS.txt", OryxSigUrl, OryxSigStableBranch())
}
func OryxSigHandshake() string {
	return fmt.Sprintf("%v(%v)", OryxSigKey, Version())
}
func OryxSigRelease() string {
	return fmt.Sprintf("%v/tree/%v", OryxSigUrl, OryxSigStableBranch())
}
func OryxSigServer() string {
	return fmt.Sprintf("%v/%v(%v)", OryxSigKey, Version(), OryxSigCode)
}
