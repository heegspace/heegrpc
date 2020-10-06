package main

import (
	"context"
	"fmt"

	"heegrpc/example/gen-go/example"

	"github.com/heegspace/heegrpc"
	"github.com/heegspace/heegrpc/registry"
	"github.com/heegspace/heegrpc/rpc"
)

func main() {
	_reg_option := rpc.Option{
		Url: "http://s2s.data.com",
	}

	registry := registry.NewRegistry()
	err := registry.Init(&_reg_option)
	if nil != err {
		panic(err)
	}

	info, err := registry.Selector("example_test")
	if nil != err {
		panic(err)
	}

	fmt.Println(info)
	client := heegrpc.NewHeegRpcClient(rpc.Option{
		Addr: info.Host,
		Port: int(info.Port),
	})

	thclient := example.NewExampleServiceClientFactory(client.Client())

	req := &example.ExampleReq{
		Reqcode:  200,
		Reqvalue: "example req",
	}

	res, err := thclient.GetResponse(context.TODO(), req)
	if nil != err {
		panic(err.Error())
	}

	fmt.Println(res)
	client.Close()
	return
}
