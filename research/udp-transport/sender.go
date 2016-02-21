package main

import (
	"flag"
	"fmt"
	ocore "github.com/ossrs/go-oryx-lib/logger"
	"net"
	"os"
)

func serve(host, transport string, port int) (err error) {
	if transport == "tcp" {
		var addr *net.TCPAddr
		if addr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", host, port)); err != nil {
			return
		}

		var c *net.TCPConn
		if c, err = net.DialTCP("tcp", nil, addr); err != nil {
			return
		}
		ocore.Trace.Println(nil, "connected at", c.RemoteAddr())
	}
	return
}

func main() {
	var host, transport string
	var port int
	flag.StringVar(&transport, "transport", "tcp", "the underlayer transport")
	flag.StringVar(&host, "host", "127.0.0.1", "the host to send to")
	flag.IntVar(&port, "port", 0, "the transport port to bind")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Usage: %s <--host=string> <--port=int> <--transport=tcp|udp>", os.Args[0]))
		fmt.Fprintln(os.Stderr, "        host, the host to send to.")
		fmt.Fprintln(os.Stderr, "        port, the transport port to bind.")
		fmt.Fprintln(os.Stderr, "        transport, the underlayer transport, tcp or udp.")
		fmt.Fprintln(os.Stderr, "For example:")
		fmt.Fprintln(os.Stderr, fmt.Sprintf("        %s --host=127.0.0.1 --port=1935 --transport=tcp", os.Args[0]))
	}
	flag.Parse()

	if port <= 0 {
		flag.Usage()
		os.Exit(1)
	}
	ocore.Trace.Println(nil, fmt.Sprintf("sender over %v://%v:%v.", transport, host, port))

	var err error
	if err = serve(host, transport, port); err != nil {
		ocore.Error.Println(nil, "serve failed. err is", err)
		os.Exit(1)
	}
}
