package zrpc

import (
	"context"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestClient_dialTimeout(t *testing.T) {
	t.Parallel()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	var f newClientFunc = func(conn net.Conn, opt *Option) (*Client, error) {
		conn.Close()
		time.Sleep(time.Second * 2)
		return nil, nil
	}
	t.Run("timeout", func(t *testing.T) {
		_, err := dialTimeout(f, "tcp", l.Addr().String(), &Option{ConnectTimeout: time.Second})
		_assert(err != nil && strings.Contains(err.Error(), "connect timeout"), "expect a timeout error")
	})
	t.Run("0", func(t *testing.T) {
		_, err := dialTimeout(f, "tcp", l.Addr().String(), &Option{ConnectTimeout: 0})
		_assert(err == nil, "0 means no limit")
	})
}

type Bar int

func (Bar) Timeout(argv int, replyv *int) error {
	time.Sleep(time.Second * 2)
	return nil
}

func startServer(t *testing.T, addr chan string) {
	var bar Bar
	Register(bar)
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
		return
	}
	addr <- l.Addr().String()
	Accept(l)
}

func TestClient_Call(t *testing.T) {
	t.Parallel()
	addrCh := make(chan string)
	go startServer(t, addrCh)
	addr := <-addrCh
	time.Sleep(time.Second)
	t.Run("client timeout", func(t *testing.T) {
		client, err := Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		var reply int
		err = client.Call(ctx, "Bar.Timeout", 1, &reply)
		_assert(err != nil && strings.Contains(err.Error(), ctx.Err().Error()), "expect a timeout error")
	})
	t.Run("server handle timeout", func(t *testing.T) {
		client, err := Dial("tcp", addr, &Option{HandleTimeout: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		var reply int
		err = client.Call(context.Background(), "Bar.Timeout", 1, &reply)
		_assert(err != nil && strings.Contains(err.Error(), "handle timeout"), "expect a timeout error")
	})
}

func TestXDial(t *testing.T) {
	if runtime.GOOS == "linux" {
		ch := make(chan struct{})
		addr := "/tmp/zrpc.sock"
		go func() {
			os.Remove(addr)
			l, err := net.Listen("unix", addr)
			if err != nil {
				log.Fatal("failed to listen unix socket")
			}
			ch <- struct{}{}
			Accept(l)
		}()
		<-ch
		_, err := XDial("unix://" + addr)
		_assert(err == nil, "fail to connect to uxix socket")
	}
}
