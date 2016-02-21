package main

import (
	"flag"
	"fmt"
	ocore "github.com/ossrs/go-oryx-lib/logger"
	"net"
	"os"
	"time"
	"encoding/json"
)

type Msg struct {
	Id uint32 `json:"id"`
	Timestamp uint64 `json:"ts"`
	Diff int `json:"diff"`
	Interval int `json:"interval"`
	Size int `json:"size"`
	Data string `json:"data"`
}

func fill_string(size int) (str string) {
	for len(str) < size {
		str += "F"
	}
	return
}

func serve_send(host, transport string, port, interval, size int) (err error) {
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

		c.SetNoDelay(true)

		var id uint32
		var prets uint64
		for {
			msg := &Msg{
				Id: id,
				Timestamp: uint64(time.Now().UnixNano()),
				Diff: 0,
				Interval: interval,
				Size: size,
				Data: "",
			}
			id++

			if prets != 0 {
				msg.Diff = (int)(msg.Timestamp - prets) / 1000 / 1000 - interval
			}
			prets = msg.Timestamp

			var buf []byte
			for {
				if buf,err = json.Marshal(msg); err != nil {
					return
				}
				if len(buf) == size {
					break
				}
				psize := size - len(buf)
				//ocore.Trace.Println(nil, "resize", len(buf), "to", size, psize)
				msg.Data = fill_string(psize)
			}

			if _,err = c.Write(buf); err != nil {
				return
			}
			ocore.Trace.Println(nil, "send", len(buf), "bytes", msg.Id, msg.Timestamp, msg.Diff, msg.Interval, msg.Size)

			time.Sleep(time.Millisecond * time.Duration(interval))
		}
	}
	return
}

func main() {
	var host, transport string
	var port, interval, size int
	flag.StringVar(&transport, "transport", "tcp", "the underlayer transport")
	flag.StringVar(&host, "host", "127.0.0.1", "the host to send to")
	flag.IntVar(&size, "size", 188, "the size of each packet data")
	flag.IntVar(&interval, "interval", 30, "the ms of send interval")
	flag.IntVar(&port, "port", 0, "the transport port to bind")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Usage: %s <--host=string> <--port=int> <--transport=tcp|udp> <--interval=int> <--size=int>", os.Args[0]))
		fmt.Fprintln(os.Stderr, "        host, the host to send to.")
		fmt.Fprintln(os.Stderr, "        port, the transport port to bind.")
		fmt.Fprintln(os.Stderr, "        transport, the underlayer transport, tcp or udp.")
		fmt.Fprintln(os.Stderr, "        interval, the ms of send interval.")
		fmt.Fprintln(os.Stderr, "        size, the size of each packet data.")
		fmt.Fprintln(os.Stderr, "For example:")
		fmt.Fprintln(os.Stderr, fmt.Sprintf("        %s --host=127.0.0.1 --port=1935 --transport=tcp --interval=30 --size=188", os.Args[0]))
	}
	flag.Parse()

	if port <= 0 {
		flag.Usage()
		os.Exit(1)
	}
	ocore.Trace.Println(nil, fmt.Sprintf("sender over %v://%v:%v.", transport, host, port))

	var err error
	if err = serve_send(host, transport, port, interval, size); err != nil {
		ocore.Error.Println(nil, "serve failed. err is", err)
		os.Exit(1)
	}
}
