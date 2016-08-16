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

package main

import "testing"

func TestPortPool_Alloc(t *testing.T) {
	p := NewPortPool(1, 10)

	if _, err := p.Alloc(0); err == nil {
		t.Error("should error")
	}

	if ps, err := p.Alloc(1); err != nil {
		t.Errorf("alloc failed, err is %v", err)
	} else if len(ps) != 1 || ps[0] != 1 {
		t.Errorf("alloc failed, ports=%v", ps)
	} else if ps, err = p.Alloc(9); err != nil {
		t.Errorf("alloc failed, err is %v", err)
	} else if len(ps) != 9 || ps[0] != 2 || ps[8] != 10 {
		t.Errorf("alloc failed, ports=%v", ps)
	} else if ps, err = p.Alloc(1); err == nil {
		t.Errorf("should error, ports=%v", ps)
	}
}

func TestPortPool_Free(t *testing.T) {
	p := NewPortPool(1, 10)

	if ps, err := p.Alloc(10); err != nil || len(ps) != 10 || ps[0] != 1 || ps[9] != 10 {
		t.Errorf("alloc failed, ports=%v, err is %v", ps, err)
	}
	p.Free(11)
	if ps, err := p.Alloc(1); err != nil || len(ps) != 1 || ps[0] != 11 {
		t.Error("free failed, ports=%v, err is %v", ps, err)
	} else if ps[0] != 11 {
		t.Errorf("invalid port=%v", ps)
	}
}

func TestPortPool_Alloc2(t *testing.T) {
	p := NewPortPool(1, 100)

	if ps, err := p.Alloc(64); err != nil || len(ps) != 64 || ps[0] != 1 || ps[63] != 64 {
		t.Errorf("alloc failed, ports=%v, err is %v", ps, err)
	} else if ps, err := p.Alloc(36); err != nil || len(ps) != 36 || ps[0] != 65 || ps[35] != 100 {
		t.Errorf("alloc failed, ports=%v, err is %v", ps, err)
	} else if ps, err = p.Alloc(1); err == nil {
		t.Errorf("should error, ports=%v", ps)
	}

	p.Free(11)
	if ps, err := p.Alloc(1); err != nil || len(ps) != 1 || ps[0] != 11 {
		t.Errorf("alloc failed, ports=%v, err is %v", ps, err)
	} else if ps, err = p.Alloc(1); err == nil {
		t.Errorf("should error, ports=%v", ps)
	}
}

func TestRetrieveVersion(t *testing.T) {
	var err error
	var ver *SrsVersion
	if ver, err = RetrieveVersion(""); err == nil {
		t.Errorf("should error")
	}
	if ver, err = RetrieveVersion("1"); err == nil {
		t.Errorf("should error")
	}
	if ver, err = RetrieveVersion("1.2"); err == nil {
		t.Errorf("should error")
	}

	if ver, err = RetrieveVersion("abc"); err == nil {
		t.Errorf("should error")
	}
	if ver, err = RetrieveVersion("1.abc"); err == nil {
		t.Errorf("should error")
	}
	if ver, err = RetrieveVersion("1.2.abc"); err == nil {
		t.Errorf("should error, ver is %v", ver)
	}

	if ver, err = RetrieveVersion("1.2.3"); err != nil {
		t.Errorf("failed, err is %v", err)
	} else if ver.Major != 1 || ver.Minor != 2 || ver.Revision != 3 || ver.Extra != 0 {
		t.Errorf("invalid, major=%v, minor=%v, revision=%v, extra=%v",
			ver.Major, ver.Minor, ver.Revision, ver.Extra)
	}

	if ver, err = RetrieveVersion("1.2.3-4"); err != nil {
		t.Errorf("failed, err is %v", err)
	} else if ver.Major != 1 || ver.Minor != 2 || ver.Revision != 3 {
		t.Errorf("invalid, major=%v, minor=%v, revision=%v",
			ver.Major, ver.Minor, ver.Revision)
	} else if ver.Extra != 4 {
		t.Errorf("invalid, extra=%v", ver.Extra)
	}
}

func TestSrsVersion_String(t *testing.T) {
	ver0 := SrsVersion{Major: 1, Minor: 2, Revision: 3}
	ver1 := SrsVersion{Major: 1, Minor: 2, Revision: 3}
	if ver0.String() != ver1.String() {
		t.Errorf("invalid")
	}

	ver1 = SrsVersion{Major: 1, Minor: 2, Revision: 4}
	if ver0.String() == ver1.String() {
		t.Errorf("invalid")
	}

	ver1 = SrsVersion{Major: 1, Minor: 4, Revision: 3}
	if ver0.String() == ver1.String() {
		t.Errorf("invalid")
	}

	ver1 = SrsVersion{Major: 4, Minor: 2, Revision: 3}
	if ver0.String() == ver1.String() {
		t.Errorf("invalid")
	}

	ver1 = SrsVersion{Major: 1, Minor: 2, Revision: 3, Extra: 4}
	if ver0.String() == ver1.String() {
		t.Errorf("invalid")
	}
}
