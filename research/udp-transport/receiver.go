package main

import (
	"flag"
	"fmt"
	ocore "github.com/ossrs/go-oryx-lib/logger"
	"net"
	"os"
	"bufio"
	"encoding/json"
	"time"
)

type Msg struct {
	Id uint32 `json:"id"`
	Timestamp uint64 `json:"ts"`
	Diff int `json:"diff"`
	Interval int `json:"interval"`
	Size int `json:"size"`
	Data string `json:"data"`
}

func serve_recv(transport string, port int) (err error) {
	if transport == "tcp" {
		var addr *net.TCPAddr
		if addr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf(":%v", port)); err != nil {
			return
		}

		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", addr); err != nil {
			return
		}
		ocore.Trace.Println(nil, "listen ok.")

		for {
			var c *net.TCPConn
			if c, err = l.AcceptTCP(); err != nil {
				return
			}
			ocore.Trace.Println(nil, "got sender", c.RemoteAddr())

			go func(c *net.TCPConn) {
				defer c.Close()

				c.SetNoDelay(true)

				br := bufio.NewReader(c)
				d := json.NewDecoder(br)

				var prets int64
				for {
					msg := &Msg{}
					if err = d.Decode(msg); err != nil {
						return
					}

					ts := time.Now().UnixNano()

					var rdiff int
					if prets != 0 {
						rdiff = (int)(ts - prets) / 1000 / 1000 - msg.Interval
					}
					prets = ts

					ocore.Trace.Println(nil, "recv", msg.Size, "bytes", msg.Id, msg.Timestamp,
						fmt.Sprintf("%v/%v", msg.Diff, rdiff), msg.Interval, msg.Size)
				}
			}(c)
		}
	}
	return
}

func main() {
	var transport string
	var port int
	flag.StringVar(&transport, "transport", "tcp", "the underlayer transport")
	flag.IntVar(&port, "port", 0, "the transport port to bind")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Usage: %s <--port=int> <--transport=tcp|udp>", os.Args[0]))
		fmt.Fprintln(os.Stderr, "        port, the transport port to bind.")
		fmt.Fprintln(os.Stderr, "        transport, the underlayer transport, tcp or udp.")
		fmt.Fprintln(os.Stderr, "For example:")
		fmt.Fprintln(os.Stderr, fmt.Sprintf("        %s --port=1935 --transport=tcp", os.Args[0]))
	}
	flag.Parse()

	if port <= 0 {
		flag.Usage()
		os.Exit(1)
	}
	ocore.Trace.Println(nil, fmt.Sprintf("receiver over %v://:%v.", transport, port))

	var err error
	if err = serve_recv(transport, port); err != nil {
		ocore.Error.Println(nil, "serve failed. err is", err)
		os.Exit(1)
	}
}
