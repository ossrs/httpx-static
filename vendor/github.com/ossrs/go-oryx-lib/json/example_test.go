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

package json_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	oj "github.com/ossrs/go-oryx-lib/json"
	"io/ioutil"
)

func ExampleJson() {
	r := bytes.NewReader([]byte(`
	{
		"code":0, // An int error code, where 0 is success.
		"data": /*An interface{} data.*/ "There is no error",
		"json+": true // Both "" and '' is ok for json+
	}
	`))

	j := json.NewDecoder(oj.NewJsonPlusReader(r))

	obj := struct {
		Code     int    `json:"code"`
		Data     string `json:"data"`
		JsonPlus bool   `json:"json+"`
	}{}
	if err := j.Decode(&obj); err != nil {
		fmt.Println("json+ decode failed, err is", err)
		return
	}

	// User can decode more objects from reader.

	fmt.Println("Code:", obj.Code)
	fmt.Println("Data:", obj.Data)
	fmt.Println("JsonPlus:", obj.JsonPlus)

	// Output:
	// Code: 0
	// Data: There is no error
	// JsonPlus: true
}

func ExampleJson_Unmarshal() {
	s := `{"code":100,"data":"There is something error"}`
	var obj interface{}
	if err := oj.Unmarshal(bytes.NewBuffer([]byte(s)), &obj); err != nil {
		fmt.Println("json+ unmarshal failed, err is", err)
		return
	}

	if obj, ok := obj.(map[string]interface{}); ok {
		fmt.Println("Code:", obj["code"])
		fmt.Println("Data:", obj["data"])
	}

	// Output:
	// Code: 100
	// Data: There is something error
}

func ExampleJson_NewCommentReader() {
	// The following is comment:
	// 		# xxxx \n
	// Where the following is string:
	//		'xxx'
	//		"xxx"
	// That is, the following is not a comment:
	//		"xxx'xx'xx#xxx"
	r := bytes.NewReader([]byte(`
# for which cannot identify the required vhost.
vhost __defaultVhost__ {
}
	`))
	cr := oj.NewCommentReader(r,
		[][]byte{[]byte("'"), []byte("\""), []byte("#")},
		[][]byte{[]byte("'"), []byte("\""), []byte("\n")},
		[]bool{false, false, true},
		[]bool{true, true, false},
	)
	if o, err := ioutil.ReadAll(cr); err != nil {
		fmt.Println("json+ read without comments failed, err is", err)
		return
	} else {
		fmt.Println(string(o))
	}

	// Output:
	// vhost __defaultVhost__ {
	// }
}
