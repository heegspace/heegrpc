package main

import (
	"context"
	"fmt"
	"heegrpc"
	"heegrpc/example/gen-go/example"
)

func main() {
	client := heegrpc.NewHeegRpcClient()
	thclient := example.NewExampleServiceClientFactory(client.Client(""))

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
