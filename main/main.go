package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
	"zrpc"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Add(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addr chan string) {
	var foo Foo
	if err := zrpc.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}
	// pick a free port
	// l, err := net.Listen("tcp", ":0")
	l, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	zrpc.HandleHTTP()
	addr <- l.Addr().String()
	// zrpc.Accept(l)
	http.Serve(l, nil)
}

func call(addr chan string) {
	// go startServer(addr)
	// client, err := zrpc.Dial("tcp", <-addr)
	// client, err := zrpc.DialHTTP("tcp", <-addr)
	client, err := zrpc.XDial("http://" + <-addr)
	if err != nil {
		log.Fatal("network error:", err)
	}
	defer client.Close()

	time.Sleep(time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			if err := client.Call(context.Background(), "Foo.Add", args, &reply); err != nil {
				log.Fatal("call Foo.Add error:", err)
			}
			log.Printf("%d + %d = %d", i, i*i, reply)
		}(i)
	}
	wg.Wait()
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go call(addr)
	startServer(addr)
}
