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

package app

import (
	"encoding/json"
	"fmt"
	"github.com/ossrs/go-oryx/core"
	"io/ioutil"
	"strings"
	"testing"
)

func TestConfigBasic(t *testing.T) {
	c := NewConfig()

	if c.Workers != 1 {
		t.Error("workers failed.")
	}

	if c.Listen != core.RtmpListen {
		t.Error("listen failed.")
	}

	if c.Go.GcInterval != 300 {
		t.Error("go gc interval failed.")
	}

	if c.Log.Tank != "file" {
		t.Error("log tank failed.")
	}

	if c.Log.Level != "trace" {
		t.Error("log level failed.")
	}

	if c.Log.File != "oryx.log" {
		t.Error("log file failed.")
	}

	if c.Heartbeat.Enabled {
		t.Error("log heartbeat enabled failed")
	}

	if c.Heartbeat.Interval != 9.3 {
		t.Error("log heartbeat interval failed")
	}

	if c.Heartbeat.Url != "http://127.0.0.1:8085/api/v1/servers" {
		t.Error("log heartbeat url failed")
	}

	if c.Heartbeat.Summary {
		t.Error("log heartbeat summary failed")
	}

	if c.Stat.Network != 0 {
		t.Error("log stat network failed")
	}
}

func BenchmarkConfigBasic(b *testing.B) {
	pc := NewConfig()
	cc := NewConfig()
	if err := pc.Reload(cc); err != nil {
		b.Error("reload failed.")
	}
}

func ExampleConfig_Loads() {
	c := NewConfig()

	//if err := c.Loads("config.json"); err != nil {
	//    panic(err)
	//}

	fmt.Println("listen at", c.Listen)
	fmt.Println("workers is", c.Workers)
	fmt.Println("go gc every", c.Go.GcInterval, "seconds")

	// Output:
	// listen at 1935
	// workers is 1
	// go gc every 300 seconds
}

func TestConfigReader(t *testing.T) {
	f := func(vs []string, eh func(string, string, string)) {
		for i := 0; i < len(vs)-1; i += 2 {
			o := vs[i]
			e := vs[i+1]

			if b, err := ioutil.ReadAll(NewReader(strings.NewReader(o))); err != nil {
				t.Error("read", o, "failed, err is", err)
			} else {
				eh(o, e, string(b))
			}
		}
		return
	}

	f([]string{
		"//comments", "",
		"/*comments*/", "",
		"//comments\nabc", "abc",
		"/*comments*/abc", "abc",
		"a/*comments*/b", "ab",
		"a//comments\nb", "ab",
	}, func(v string, e string, o string) {
		if e != o {
			t.Error("for", v, "expect", len(e), "size", e, "but got", len(o), "size", o)
		}
	})
}

func TestConfigComments(t *testing.T) {
	f := func(vs []string, eh func(string, interface{}, error)) {
		for _, v := range vs {
			j := json.NewDecoder(NewReader(strings.NewReader(v)))
			var o interface{}
			err := j.Decode(&o)
			eh(v, o, err)
		}
	}

	f([]string{
		`
        {
            // the RTMP listen port.
            "listen": 1935,
            // whether start in daemon for unix-like os.
            "daemon": false,
            /**
            * the go runtime config.
            * for go-oryx specified.
            */
            "go": {
                "gc_interval": 300,
                "max_threads": 0 // where 0 is use default.
            }
        }
        `,
	}, func(v string, o interface{}, err error) {
		if err != nil {
			t.Error("show pass for", v, "actual err is", err)
		}
	})

	f([]string{
		"{}//empty",
		"{}/*empty*/",

		`//c++ style
        {"listen": 1935}`,

		`/*c style*/
        {"listen": 1935}`,

		`/*c style*/{"listen": 1935}`,

		`//c++ style
        {"listen": 1935}
        //c++ style`,

		`/*c style*/
        {"listen": 1935}/*c style*/`,

		`/*c style*/ {"listen": /* c style */1935}`,

		`{"url": "http://server/api"}`,
	}, func(v string, o interface{}, err error) {
		if err != nil {
			t.Error("show pass for", v, "actual err is", err)
		}
	})

	f([]string{
		`{"listen": 1935}`,
		`{"listen": 1935, "daemon": true}`,
	}, func(v string, o interface{}, err error) {
		if err != nil {
			t.Error("show pass for", v, "actual err is", err)
		}
	})

	f([]string{
		"/*comments",
		`{"listen":1935/*comments}`,
	}, func(v string, o interface{}, err error) {
		if err == nil {
			t.Error("show failed for", v)
		}
	})
}
