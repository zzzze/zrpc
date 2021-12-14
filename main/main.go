package main

import (
	"context"
	"log"
	"net"
	"sync"
	"time"
	"zrpc"
	"zrpc/xclient"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Add(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (f Foo) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Num1))
	*reply = args.Num1 + args.Num2
	return nil
}

// func startServer(addr chan string) {
// 	var foo Foo
// 	if err := zrpc.Register(&foo); err != nil {
// 		log.Fatal("register error:", err)
// 	}
// 	l, err := net.Listen("tcp", ":9999")
// 	if err != nil {
// 		log.Fatal("network error:", err)
// 	}
// 	log.Println("start rpc server on", l.Addr())
// 	zrpc.HandleHTTP()
// 	http.Serve(l, nil)
// }

// func startServer(addr chan string) {
// 	var foo Foo
// 	if err := zrpc.Register(&foo); err != nil {
// 		log.Fatal("register error:", err)
// 	}
// 	// pick a free port
// 	l, err := net.Listen("tcp", ":0")
// 	if err != nil {
// 		log.Fatal("network error:", err)
// 	}
// 	log.Println("start rpc server on", l.Addr())
// 	addr <- l.Addr().String()
// 	zrpc.Accept(l)
// }

func startServer(addr chan string) {
	var foo Foo
	// pick a free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	server := zrpc.NewServer()
	if err := server.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	server.Accept(l)
}

// func call(addr chan string) {
// 	// go startServer(addr)
// 	// client, err := zrpc.Dial("tcp", <-addr)
// 	// client, err := zrpc.DialHTTP("tcp", <-addr)
// 	client, err := zrpc.XDial("http://" + <-addr)
// 	if err != nil {
// 		log.Fatal("network error:", err)
// 	}
// 	defer client.Close()
//
// 	time.Sleep(time.Second)
// 	var wg sync.WaitGroup
// 	for i := 0; i < 5; i++ {
// 		wg.Add(1)
// 		go func(i int) {
// 			defer wg.Done()
// 			args := &Args{Num1: i, Num2: i * i}
// 			var reply int
// 			if err := client.Call(context.Background(), "Foo.Add", args, &reply); err != nil {
// 				log.Fatal("call Foo.Add error:", err)
// 			}
// 			log.Printf("%d + %d = %d", i, i*i, reply)
// 		}(i)
// 	}
// 	wg.Wait()
// }

func foo(ctx context.Context, c *xclient.XClient, typ, serviceMethod string, args *Args) {
	var reply int
	var err error
	switch typ {
	case "call":
		err = c.Call(ctx, serviceMethod, args, &reply)
	case "broadcast":
		err = c.Broadcast(ctx, serviceMethod, args, &reply)
	}
	if err != nil {
		log.Printf("%s %s error: %v", typ, serviceMethod, err)
	} else {
		log.Printf("%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
	}
}

func call(addr1, addr2 string) {
	d := xclient.NewMultiServersDiscovery([]string{
		"tcp://" + addr1,
		"tcp://" + addr2,
	})
	c := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer c.Close()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(context.Background(), c, "call", "Foo.Add", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func broadcast(addr1, addr2 string) {
	d := xclient.NewMultiServersDiscovery([]string{
		"tcp://" + addr1,
		"tcp://" + addr2,
	})
	c := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer c.Close()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(context.Background(), c, "broadcast", "Foo.Add", &Args{Num1: i, Num2: i * i})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			foo(ctx, c, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
			cancel()
		}(i)
	}
	wg.Wait()
}

func main() {
	log.SetFlags(0)
	ch1 := make(chan string)
	ch2 := make(chan string)
	go startServer(ch1)
	go startServer(ch2)
	addr1 := <-ch1
	addr2 := <-ch2

	time.Sleep(time.Second)
	call(addr1, addr2)
	broadcast(addr1, addr2)
}
