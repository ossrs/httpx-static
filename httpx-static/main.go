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

/*
 This the main entrance of https-proxy, proxy to api or other http server.
*/
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/ossrs/go-oryx-lib/https"
	"net/http"
	"os"
	"strings"
)

func main() {
	var httpPort, httpsPort int
	flag.IntVar(&httpPort, "http", 80, "http listen at. 0 to disable http.")
	flag.IntVar(&httpsPort, "https", 443, "https listen at. 0 to disable https. 443 to serve.")
	flag.Parse()

	if httpsPort != 0 && httpsPort != 443 {
		fmt.Println("https must be 0(disabled) or 443(enabled)")
		os.Exit(-1)
	}
	if httpPort == 0 && httpsPort == 0 {
		fmt.Println("http or https are disabled")
		os.Exit(-1)
	}

	html := http.Dir("./html")
	fh := http.FileServer(html)
	http.Handle("/", fh)

	var err error
	var m https.Manager
	if m, err = https.NewLetsencryptManager("", nil, "letsencrypt.cache"); err != nil {
		fmt.Println("https failed, err is", err)
		return
	}

	var protos []string
	if httpPort != 0 {
		protos = append(protos, fmt.Sprintf("http(:%v)", httpPort))
	}
	if httpsPort != 0 {
		protos = append(protos, fmt.Sprintf("https(:%v)", httpsPort))
	}
	fmt.Println(fmt.Sprintf("%v html root at %v", strings.Join(protos, ", "), string(html)))

	go func() {
		if err := http.ListenAndServe(":http", nil); err != nil {
			panic(err)
		}
	}()

	svr := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: m.GetCertificate,
		},
	}

	if err := svr.ListenAndServeTLS("", ""); err != nil {
		panic(err)
	}
}
