package main

import (
	"context"
	"log"
	"net"
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
	client, err := zrpc.Dial("tcp", <-addr)
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
