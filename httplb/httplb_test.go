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

import (
	"net/http"
	"net/url"
	"testing"
)

func TestHlsPlusProxy(t *testing.T) {
	q, h := url.Values{}, http.Header{}

	proxy := NewHlsPlusProxy(nil)
	vconn, _, err := proxy.identify(q, h, "", 0)
	if err == nil {
		t.Errorf("should failed.")
	}
	q.Set("shp_xpsid", "0381u1odj28371jso1823j3o1")
	h.Set("X-Playback-Session-Id", "0381u1odj28371jso1823j3o1")
	if vconn, _, err = proxy.identify(q, h, "", 0); err == nil {
		t.Errorf("should failed.")
	}

	q, h = url.Values{}, http.Header{}
	if vconn, _, err = proxy.identify(q, h, "127.0.0.1:1234", 0); err != nil {
		t.Error("failed, err is", err)
	} else if len(vconn.addrs) != 1 {
		t.Errorf("invalid addrs=%v", len(vconn.addrs))
	}

	proxy = NewHlsPlusProxy(nil)
	q.Set("shp_xpsid", "0381u1odj28371jso1823j3o1")
	if vconn, _, err = proxy.identify(q, h, "127.0.0.1:1234", 0); err != nil {
		t.Error("failed, err is", err)
	} else if len(vconn.addrs) != 1 {
		t.Errorf("invalid addrs=%v", len(vconn.addrs))
	} else if vconn.xpsid != "0381u1odj28371jso1823j3o1" {
		t.Error("invalid xpsid=%v", vconn.xpsid)
	}

	proxy = NewHlsPlusProxy(nil)
	h.Set("X-Playback-Session-Id", "0381u1odj28371jso1823j3o1")
	if vconn, _, err = proxy.identify(q, h, "127.0.0.1:1234", 0); err != nil {
		t.Error("failed, err is", err)
	} else if len(vconn.addrs) != 1 {
		t.Errorf("invalid addrs=%v", len(vconn.addrs))
	} else if vconn.xpsid != "0381u1odj28371jso1823j3o1" {
		t.Error("invalid xpsid=%v", vconn.xpsid)
	}

	proxy = NewHlsPlusProxy(nil)
	q.Set("shp_uuid", "0381u1odj28371jso1823j3o1")
	if vconn, _, err = proxy.identify(q, h, "127.0.0.1:1234", 0); err != nil {
		t.Error("failed, err is", err)
	} else if len(vconn.addrs) != 1 {
		t.Errorf("invalid addrs=%v", len(vconn.addrs))
	} else if vconn.uuid != "0381u1odj28371jso1823j3o1" {
		t.Errorf("invalid uuid=%v", vconn.uuid)
	}
	if c, _, err := proxy.identify(q, h, "127.0.0.1:1234", 0); err != nil {
		t.Error("failed, err is", err)
	} else if c != vconn {
		t.Error("invalid conn")
	}
}
