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

package core

import "fmt"

const (
	major     = 0
	minor     = 1
	reversion = 17
)

// Version returns the go-oryx major.minor.revision version
func Version() string {
	return fmt.Sprintf("%v.%v.%v", major, minor, reversion)
}

// OryxSigStable stable major version
const OryxSigStable = 0

// OryxSigStableBranch returns a go-oryx stable branch string
func OryxSigStableBranch() string {
	return fmt.Sprintf("%v.0release", OryxSigStable)
}

// project info.

// OryxSigKey specifies the project key
const OryxSigKey = "Oryx"

// OryxSigCode specifies the project code
const OryxSigCode = "MonkeyKing"

// OryxSigRole specifies the project role
const OryxSigRole = "cluster"

// OryxSigName specifies the project name
const OryxSigName = OryxSigKey + "(SRS++)"

// OryxSigURLShort specifies the short project URL
const OryxSigURLShort = "github.com/ossrs/go-oryx"

// OryxSigURL specifies the full project URL
const OryxSigURL = "https://" + OryxSigURLShort

// OryxSigWeb specifies the project website
const OryxSigWeb = "http://ossrs.net"

// OryxSigEmail specifies the project owner's email address
const OryxSigEmail = "winlin@vip.126.com"

// OryxSigLicense specifies the product:  go-oryx/SRS++
const OryxSigLicense = "The MIT License (MIT)"

// OryxSigCopyright specifies the product:  go-oryx/SRS++
const OryxSigCopyright = "Copyright (c) 2013-2015 Oryx(ossrs)"

// OryxSigAuthors specifies the project authors
const OryxSigAuthors = "winlin"

// OryxSigProduct specifies the project description
const OryxSigProduct = "The go-oryx is SRS++, focus on real-time live streaming cluster."

// OryxSigPrimary returns the primary stable major version
func OryxSigPrimary() string {
	return fmt.Sprintf("Oryx/%v", OryxSigStable)
}

// OryxSigContributorsURL returns the formatted contributors URL
func OryxSigContributorsURL() string {
	return fmt.Sprintf("%v/blob/%v/AUTHORS.txt", OryxSigURL, OryxSigStableBranch())
}

// OryxSigHandshake returns the formatted handshake built from product key and version
func OryxSigHandshake() string {
	return fmt.Sprintf("%v(%v)", OryxSigKey, Version())
}

// OryxSigRelease returns the formatted release URL
func OryxSigRelease() string {
	return fmt.Sprintf("%v/tree/%v", OryxSigURL, OryxSigStableBranch())
}

// OryxSigServer returns the full, formatted server version information
func OryxSigServer() string {
	return fmt.Sprintf("%v/%v(%v)", OryxSigKey, Version(), OryxSigCode)
}
