package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
	"zrpc"
	"zrpc/codec"
)

func startServer(addr chan string) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	zrpc.Accept(l)
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	conn, err := net.Dial("tcp", <-addr)
	if err != nil {
		log.Fatal("network error:", err)
	}
	defer conn.Close()

	time.Sleep(time.Second)
	err = json.NewEncoder(conn).Encode(zrpc.DefaultOption)
	if err != nil {
		log.Fatal("write option to conn error:", err)
	}
	cc := codec.NewGobCodec(conn)
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		if err := cc.Write(h, fmt.Sprintf("zrpc req %d", h.Seq)); err != nil {
			log.Fatal("write to conn error:", err)
		}
		if err := cc.ReadHeader(h); err != nil {
			log.Fatal("read header error:", err)
		}
		var reply string
		if err := cc.ReadBody(&reply); err != nil {
			log.Fatal("read body error:", err)
		}
		log.Println("reply:", reply)
	}
}
