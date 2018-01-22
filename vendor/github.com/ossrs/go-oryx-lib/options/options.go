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

// The oryx options package parse the options with config file.
//		-c, to specifies the config file.
//		-v, to print version and quit.
//		-g, to print signature and quit.
//		-h, to print help and quit.
// We return the parsed config file path.
package options

import (
	"flag"
	"fmt"
	"os"
)

// parse the argv with config, version and signature.
// @param rcf The recomment config file path.
// @param version The vesion of application, such as 1.2.3
// @param signature The signature of application, such as SRS/1.2.3
func ParseArgv(rcf, version, signature string) (confFile string) {
	// the args format:
	//          -c conf/ory.json
	//          --c conf/oryx.json
	//          -c=conf/oryx.json
	//          --c=conf/oryx.json
	//          --conf=conf/oryx.json
	if true {
		dv := ""
		ua := "The config file"
		flag.StringVar(&confFile, "c", dv, ua)
		flag.StringVar(&confFile, "conf", dv, ua)
	}

	var showVersion bool
	if true {
		dv := false
		ua := "Print version"
		flag.BoolVar(&showVersion, "v", dv, ua)
		flag.BoolVar(&showVersion, "V", dv, ua)
		flag.BoolVar(&showVersion, "version", dv, ua)
	}

	var showSignature bool
	if true {
		dv := false
		ua := "print signature"
		flag.BoolVar(&showSignature, "g", dv, ua)
		flag.BoolVar(&showSignature, "signature", dv, ua)
	}

	flag.Usage = func() {
		fmt.Println(signature)
		fmt.Println(fmt.Sprintf("Usage: %v [-c|--conf <filename>] [-?|-h|--help] [-v|-V|--version] [-g|--signature]", os.Args[0]))
		fmt.Println(fmt.Sprintf("	    -c, --conf filename     : The config file path"))
		fmt.Println(fmt.Sprintf("	    -?, -h, --help          : Show this help and exit"))
		fmt.Println(fmt.Sprintf("	    -v, -V, --version       : Print version and exit"))
		fmt.Println(fmt.Sprintf("	    -g, --signature         : Print signature and exit"))
		fmt.Println(fmt.Sprintf("For example:"))
		fmt.Println(fmt.Sprintf("	    %v -c %v", os.Args[0], rcf))
	}
	flag.Parse()

	if showVersion {
		fmt.Fprintln(os.Stderr, version)
		os.Exit(0)
	}

	if showSignature {
		fmt.Fprintln(os.Stderr, signature)
		os.Exit(0)
	}

	if len(confFile) == 0 {
		flag.Usage()
		os.Exit(-1)
	}

	return
}
