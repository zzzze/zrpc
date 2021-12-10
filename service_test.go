package zrpc

import (
	"fmt"
	"reflect"
	"testing"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Add(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

// not a exported method
func (f Foo) add(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assertion failed: "+msg, v...))
	}
}

func TestNewService(t *testing.T) {
	var foo Foo
	service := newService(&foo)
	_assert(len(service.method) == 1, "wrong service Method, expect 1, but got %d", len(service.method))
}

func TestMethodType_Call(t *testing.T) {
	var foo Foo
	service := newService(&foo)
	mType := service.method["Add"]
	argv := mType.newArgv()
	argv.Set(reflect.ValueOf(Args{Num1: 10, Num2: 11}))
	replyv := mType.newReplyv()
	err := service.call(mType, argv, replyv)
	_assert(err == nil, "failed to call Foo.Add: err should be nil, but got %q", err)
	_assert(*replyv.Interface().(*int) == 21, "failed to call Foo.Add: reply should be 21, but got %d", *replyv.Interface().(*int))
	_assert(mType.NumCalls() == 1, "failed to call Foo.Add: function call count should be 1, but got %d", mType.NumCalls())
}
