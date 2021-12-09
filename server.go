package zrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"time"
	"zrpc/codec"
)

const MagicNumber = 0xd4ad87

type Option struct {
	MagicNumber int
	CodecType   codec.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber, // MagicNumber marks this's a zrpc request
	CodecType:   codec.GobType,
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("[rpc server]: accept error:", err)
			return
		}
		go server.ServeConn(conn)
	}
}

// ServeConn runs the server on a single connection.
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer conn.Close()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("[rpc server]: decode options error:", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("[rpc server]: invalid magic number %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("[rpc server]: invalid codec type %q", opt.CodecType)
		return
	}
	server.serveRequests(f(conn))
}

var invalidRequest = struct{}{}

// serveRequests read and handle requests until the connection is closed.
// serveRequests has 3 processes
// 1. readRequest
// 2. handleRequest
// 3. sendResponse
func (server *Server) serveRequests(cc codec.Codec) {
	defer cc.Close()
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break // it's not possible to recovery, so close the connection
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("[rpc server]: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// TODO: use correct type
	req.argv = reflect.New(reflect.TypeOf(""))
	if err := cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("[rpc server]: read argv err:", err)
		return req, err
	}
	return req, nil
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	// TODO: invoke method specified in req.h, and set value to req.replyv
	req.replyv = reflect.ValueOf(fmt.Sprintf("zrpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	time.Sleep(time.Second)
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("[rpc server]: write response error:", err)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}
