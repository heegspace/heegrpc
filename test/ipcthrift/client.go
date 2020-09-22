package main

import (
	"context"
	"fmt"
	"ipcthrift/ipcshared"

	"github.com/apache/thrift/lib/go/thrift"
)

func main() {
	var transport thrift.TTransport
	var protocolFactory thrift.TProtocolFactory
	var transportFactory thrift.TTransportFactory

	protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory = thrift.NewTBufferedTransportFactory(8192)
	transportFactory = thrift.NewTFramedTransportFactory(transportFactory)

	transport, err := thrift.NewTSocket("localhost:9990")

	transport, err = transportFactory.GetTransport(transport)
	if err != nil {
		panic(err.Error())
	}
	defer transport.Close()
	if err := transport.Open(); err != nil {
		panic(err.Error())
	}

	client := ipcshared.NewSharedServiceClientFactory(transport, protocolFactory)
	res, err := client.GetStruct(context.TODO(), 12)
	if nil != err {
		panic(err.Error())
	}

	fmt.Println(res)
	return
}
