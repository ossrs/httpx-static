package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	ocore "github.com/ossrs/go-oryx-lib/logger"
	"net"
	"os"
	"time"
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

func fill_string(size int) (str string) {
	for len(str) < size {
		str += "F"
	}
	return
}

func create_raw_message(interval, size int, id uint32, prets uint64) (uint32, uint64, *Msg, []byte, error) {
	msg := &Msg{
		Id:        id,
		Timestamp: uint64(time.Now().UnixNano()),
		Diff:      0,
		Interval:  uint32(interval),
		Size:      uint32(size),
		Data:      "",
	}
	id++

	if prets != 0 {
		msg.Diff = (int32)(msg.Timestamp-prets)/1000/1000 - int32(interval)
	}
	prets = msg.Timestamp

	var err error
	var buf []byte
	for {
		if buf, err = json.Marshal(msg); err != nil {
			return id, prets, msg, buf, err
		}
		if len(buf) == size {
			break
		}
		psize := size - len(buf)
		//ocore.Trace.Println(nil, "resize", len(buf), "to", size, psize)
		msg.Data = fill_string(psize)
	}

	return id, prets, msg, buf, nil
}

func serve_send(host, transport string, port, interval, size int, report uint32) (err error) {
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

		br := bufio.NewReader(c)
		d := json.NewDecoder(br)

		var id uint32
		var prets uint64
		for {
			var msg *Msg
			var buf []byte
			if id, prets, msg, buf, err = create_raw_message(interval, size, id, prets); err != nil {
				return
			}

			if _, err = c.Write(buf); err != nil {
				return
			}
			ocore.Info.Println(nil, "send", len(buf), "bytes",
				fmt.Sprintf("%v/%v/%v", msg.Id, msg.Timestamp, msg.Diff),
				fmt.Sprintf("%v/%v/%v", msg.Type, msg.Interval, msg.Size))

			// requires report every some messages.
			if (id % report) == 0 {
				msg.Type = MsgTypeReport
				msg.Data = ""
				if buf, err = json.Marshal(msg); err != nil {
					return
				}
				if _, err = c.Write(buf); err != nil {
					return
				}
				if err = d.Decode(msg); err != nil {
					return
				}

				m := &Metric{}
				if err = json.Unmarshal([]byte(msg.Data), m); err != nil {
					return
				}
				fmt.Fprintln(os.Stderr, fmt.Sprintf("Report start:%v duration:%v total:%v drop:%v latency:%v",
					m.Starttime/1000/1000, m.Duration, msg.Id, m.DropFrames, m.Latency))
			}

			time.Sleep(time.Millisecond * time.Duration(interval))
		}
	}
	return
}

func main() {
	var host, transport string
	var port, interval, size, report int
	flag.StringVar(&transport, "transport", "tcp", "the underlayer transport")
	flag.StringVar(&host, "host", "127.0.0.1", "the host to send to")
	flag.IntVar(&size, "size", 188, "the size of each packet data")
	flag.IntVar(&report, "report", 100, "report when got these packets")
	flag.IntVar(&interval, "interval", 300, "the ms of send interval")
	flag.IntVar(&port, "port", 0, "the transport port to bind")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Usage: %s <--host=string> <--port=int> <--transport=tcp|udp> <--interval=uint> <--size=int> <--report=uint>", os.Args[0]))
		fmt.Fprintln(os.Stderr, "        host, the host to send to.")
		fmt.Fprintln(os.Stderr, "        port, the transport port to bind.")
		fmt.Fprintln(os.Stderr, "        transport, the underlayer transport, tcp or udp.")
		fmt.Fprintln(os.Stderr, "        interval, the ms of send interval.")
		fmt.Fprintln(os.Stderr, "        size, the size of each packet data.")
		fmt.Fprintln(os.Stderr, "        report, report when got these packets.")
		fmt.Fprintln(os.Stderr, "For example:")
		fmt.Fprintln(os.Stderr, fmt.Sprintf("        %s --host=127.0.0.1 --port=1935 --transport=tcp --interval=30 --size=188 --report=100", os.Args[0]))
	}
	flag.Parse()

	if port <= 0 {
		flag.Usage()
		os.Exit(1)
	}
	ocore.Trace.Println(nil, fmt.Sprintf("sender over %v://%v:%v %v/%v/%v.", transport, host, port, interval, size, report))

	var err error
	if err = serve_send(host, transport, port, interval, size, uint32(report)); err != nil {
		ocore.Error.Println(nil, "serve failed. err is", err)
		os.Exit(1)
	}
}
