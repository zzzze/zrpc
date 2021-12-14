package xclient

import (
	"context"
	"io"
	"reflect"
	"sync"
	"zrpc"
)

type XClient struct {
	d       Discovery
	mode    SelectMode
	opt     *zrpc.Option
	mu      sync.Mutex
	clients map[string]*zrpc.Client
}

func NewXClient(d Discovery, mode SelectMode, opt *zrpc.Option) *XClient {
	return &XClient{
		d:       d,
		mode:    mode,
		opt:     opt,
		clients: make(map[string]*zrpc.Client),
	}
}

var _ io.Closer = (*XClient)(nil)

func (c *XClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, client := range c.clients {
		// I have no idea how to deal with error, just ignore it.
		_ = client.Close()
		delete(c.clients, key)
	}
	return nil
}

func (c *XClient) dial(rpcAddr string) (*zrpc.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	client, ok := c.clients[rpcAddr]
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(c.clients, rpcAddr)
		client = nil
	}
	if client != nil {
		return client, nil
	}
	client, err := zrpc.XDial(rpcAddr, c.opt)
	if err != nil {
		return nil, err
	}
	c.clients[rpcAddr] = client
	return client, nil
}

func (c *XClient) call(ctx context.Context, rpcAddr string, serviceMethod string, args, reply interface{}) error {
	client, err := c.dial(rpcAddr)
	if err != nil {
		return err
	}
	return client.Call(ctx, serviceMethod, args, reply)
}

func (c *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	rpcAddr, err := c.d.Get(c.mode)
	if err != nil {
		return err
	}
	return c.call(ctx, rpcAddr, serviceMethod, args, reply)
}

func (c *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	services, err := c.d.GetAll()
	if err != nil {
		return err
	}
	var (
		wg sync.WaitGroup
		mu sync.Mutex
		e  error
	)
	replyDone := reply == nil
	ctx, cancel := context.WithCancel(ctx)
	for _, rpcAddr := range services {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var clonedReply interface{}
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			err := c.call(ctx, rpcAddr, serviceMethod, args, clonedReply)
			mu.Lock()
			if err != nil && e == nil {
				e = err
				cancel()
			}
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	cancel()
	return e
}
