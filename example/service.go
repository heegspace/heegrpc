package main

import (
	"context"
	"fmt"
	"time"

	"heegrpc/example/gen-go/example"

	"github.com/heegspace/heegrpc"
	"github.com/heegspace/heegrpc/rpc"
	"github.com/heegspace/heegrpc/utils"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/heegspace/heegrpc/registry"
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
	addr, err := utils.ExternalIP()
	if nil != err {
		panic(err)
	}

	service := heegrpc.NewHeegRpcServer(rpc.Option{
		Addr: addr.String(),
		Port: 9099,
	})

	err = service.Init()
	if nil != err {
		panic(err.Error())
	}

	_reg_option := rpc.Option{
		Addr:    "192.168.1.4",
		Port:    9099,
		S2sName: "BADC-76DA-765E-9000-BBA7",
		Url:     "http://s2s.data.com",
	}

	registry := registry.NewRegistry()
	err = registry.Init(&_reg_option)
	if nil != err {
		panic(err)
	}

	err = registry.Register()
	if nil != err {
		panic(err)
	}

	service.Processor(NewProcessor())
	if err = service.Run(); nil != err {
		fmt.Println("Err to here")
		panic(err.Error())
	}

	return
}
