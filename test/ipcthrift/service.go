package main

import (
	"context"
	"fmt"
	"ipcthrift/ipcshared"
	"time"

	"heegrpc"

	"github.com/apache/thrift/lib/go/thrift"
	// "github.com/apache/thrift/lib/go/thrift"
)

type SharedServiceHandle struct{}

func (p *SharedServiceHandle) GetStruct(ctx context.Context, key int32) (*ipcshared.SharedStruct, error) {
	fmt.Print("getStruct(", key, ")\n")
	v := &ipcshared.SharedStruct{
		Key:   key,
		Value: "thrift test",
	}

	time.Sleep(5 * time.Second)
	return v, nil
}

func NewSharedServiceHandle() *SharedServiceHandle {
	v := &SharedServiceHandle{}

	return v
}

func NewProcessor() thrift.TProcessor {
	handler := NewSharedServiceHandle()
	processor := ipcshared.NewSharedServiceProcessor(handler)

	return processor
}

func main() {
	// var transport thrift.TServerTransport
	// var protocolFactory thrift.TProtocolFactory
	// var transportFactory thrift.TTransportFactory

	// protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	// transportFactory = thrift.NewTBufferedTransportFactory(8192)
	// transportFactory = thrift.NewTFramedTransportFactory(transportFactory)
	// transport, err := thrift.NewTServerSocket("localhost:9990")
	// if nil != err {
	// 	panic(err.Error())
	// }

	// fmt.Printf("%T\n", transport)

	// server := thrift.NewTSimpleServer4(NewProcessor(), transport, transportFactory, protocolFactory)

	// fmt.Println("Starting the simple server... on ", "localhost:9990")
	// err = server.Serve()
	// if nil != err {
	// 	panic(err.Error())
	// }
	service := heegrpc.NewHeegRpcServer()
	err := service.Init()
	if nil != err {
		panic(err.Error())
	}

	service.Processor(NewProcessor())
	if err = service.Run(); nil != err {
		panic(err.Error())
	}

	return
}
