package main

import (
	"context"
	"fmt"
	"time"

	"heegrpc/example/gen-go/example"

	"heegrpc"

	"github.com/apache/thrift/lib/go/thrift"
)

type ExampleServiceHandle struct{}

func (p *ExampleServiceHandle) GetResponse(ctx context.Context, req *example.ExampleReq) (*example.ExampleRes, error) {
	v := &example.ExampleRes{
		Rescode:  req.Reqcode,
		Resvalue: "thrift test",
		Map1:     make(map[string]string),
	}
	v.Map1["key1"] = "value1"
	v.Map1["key2"] = "value2"

	time.Sleep(5 * time.Second)
	return v, nil
}

func NewExampleServiceHandle() *ExampleServiceHandle {
	v := &ExampleServiceHandle{}

	return v
}

func NewProcessor() thrift.TProcessor {
	handler := NewExampleServiceHandle()
	processor := example.NewExampleServiceProcessor(handler)

	return processor
}

func main() {
	service := heegrpc.NewHeegRpcServer()
	err := service.Init()
	if nil != err {
		panic(err.Error())
	}

	service.Processor(NewProcessor())
	if err = service.Run(); nil != err {
		fmt.Println("Err to here")
		panic(err.Error())
	}

	return
}
