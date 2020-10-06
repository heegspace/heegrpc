package main

import (
	"context"
	"fmt"
	"strconv"

	"heegrpc/example/gen-go/example"

	"github.com/heegspace/heegrpc"
	"github.com/heegspace/heegrpc/rpc"
	"github.com/heegspace/heegrpc/utils"

	"github.com/heegspace/heegrpc/registry"
	"github.com/heegspace/thrift"
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

	registry := registry.NewRegistry()
	service := heegrpc.NewHeegRpcServer(rpc.Option{
		Addr: addr.String(),
		Port: 8088,
		ListenFunc: func(addr, port string) {
			fmt.Println("Listen: ", addr, "  ", port)
			p, _ := strconv.Atoi(port)
			_reg_option := rpc.Option{
				Addr:    addr,
				Port:    p,
				S2sName: "example_test",
				Url:     "http://s2s.data.com",
			}

			err = registry.Init(&_reg_option)
			if nil != err {
				panic(err)
			}

			err = registry.Register()
			if nil != err {
				panic(err)
			}

			fmt.Println("Listen: ", addr, "  ", port)
		},
	})

	err = service.Init()
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
