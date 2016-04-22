package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	ocore "github.com/ossrs/go-oryx-lib/logger"
)

type MsgType uint8

const (
	MsgTypeRaw MsgType = iota
	MsgTypeReport
	MsgTypeUnknown
)

func (v MsgType) String() string {
	switch v {
	case MsgTypeRaw:
		return "Raw"
	case MsgTypeReport:
		return "Report"
	default:
		return "Unknown"
	}
}

type Msg struct {
	Id        uint32  `json:"id"`
	Timestamp uint64  `json:"ts"`
	Diff      int32   `json:"diff"`
	Interval  uint32  `json:"interval"`
	Size      uint32  `json:"size"`
	Type      MsgType `json:"type"`
	Data      string  `json:"data"`
}

type Metric struct {
	Starttime  int64 `json:"start"`
	Duration   int32 `json:"duration"` // in ms.
	DropFrames int32 `json:"drop"`
	Latency    int32 `json:"latency"`
	JumpFrames int32 `json:"jump"`
}

func serve_msgs(rmsg func() (*Msg, error), wbuf func([]byte) error) (err error) {
	var prets int64
	var preid uint32
	missing := map[uint32]bool{}
	var metric *Metric
	var doResetMetric bool
	for {
		ts := time.Now().UnixNano()

		var msg *Msg
		if msg, err = rmsg(); err != nil {
			return
		}

		for i := preid + 1; i < msg.Id; i++ {
			missing[i] = true
		}
		delete(missing, msg.Id)

		if msg.Type == MsgTypeReport {
			metric.Duration = int32((ts - metric.Starttime) / 1000)

			var buf []byte
			if buf, err = json.Marshal(metric); err != nil {
				return
			}
			doResetMetric = true

			msg.Data = string(buf)
			if buf, err = json.Marshal(msg); err != nil {
				return
			}
			if err = wbuf(buf); err != nil {
				return
			}
			continue
		}

		if msg.Id > preid+1 {
			metric.JumpFrames += int32(msg.Id - (preid + 1))
		}
		if preid < msg.Id {
			preid = msg.Id
		}

		if doResetMetric || metric == nil {
			doResetMetric = false
			metric = &Metric{
				Starttime: ts,
			}
		}

		var rdiff int32
		if prets != 0 {
			rdiff = (int32)(ts-prets)/1000/1000 - int32(msg.Interval)
		}
		prets = ts

		// update metric for raw message.
		if prets > 0 && rdiff > 0 {
			metric.Latency += rdiff
		}
		metric.DropFrames = int32(len(missing))

		ocore.Info.Println(nil, "recv", msg.Size, "bytes",
			fmt.Sprintf("%v/%v", msg.Id, msg.Timestamp),
			fmt.Sprintf("%v/%v", msg.Diff, rdiff),
			fmt.Sprintf("%v/%v/%v", msg.Type, msg.Interval, msg.Size))
	}

	return
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

				_ = serve_msgs(func() (msg *Msg, err error) {
					msg = &Msg{}
					if err = d.Decode(msg); err != nil {
						return
					}
					return
				}, func(buf []byte) (err error) {
					if _, err = c.Write(buf); err != nil {
						return
					}
					return
				})
			}(c)
		}
	}

	if transport == "udp" {
		var addr *net.UDPAddr
		if addr, err = net.ResolveUDPAddr("udp", fmt.Sprintf(":%v", port)); err != nil {
			return
		}

		var c *net.UDPConn
		if c, err = net.ListenUDP("udp", addr); err != nil {
			return
		}
		ocore.Trace.Println(nil, "listen ok.")

		rbuf := make([]byte, 32*1024)
		var from *net.UDPAddr
		return serve_msgs(func() (msg *Msg, err error) {
			var n int
			if n, from, err = c.ReadFromUDP(rbuf); err != nil {
				return
			}

			msg = &Msg{}
			if err = json.Unmarshal(rbuf[:n], msg); err != nil {
				return
			}

			return
		}, func(buf []byte) (err error) {
			if _, err = c.WriteToUDP(buf, from); err != nil {
				return
			}
			return
		})
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
		fmt.Fprintln(os.Stderr, fmt.Sprintf("        %s --port=1935 --transport=udp", os.Args[0]))
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
