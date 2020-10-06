package rpc

import (
	"fmt"
	"strings"
	"time"

	"github.com/heegspace/thrift"
)

type HeegServer struct {
	server *thrift.TSimpleServer

	transport        thrift.TServerTransport
	protocolFactory  thrift.TProtocolFactory
	transportFactory thrift.TTransportFactory

	Option Option

	//
	inited    bool
	processor thrift.TProcessor
}

func NewHeegServer(option Option) *HeegServer {
	v := &HeegServer{
		server: nil,
		inited: false,
		Option: option,
	}

	return v
}

func (this *HeegServer) Init() (err error) {
	if this.inited {
		return
	}

	this.protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	this.transportFactory = thrift.NewTBufferedTransportFactory(8192)
	this.transportFactory = thrift.NewTFramedTransportFactory(this.transportFactory)

	this.transport, err = thrift.NewTServerSocket(this.Option.Bind())
	if nil != err {
		return
	}

	this.inited = true
	return
}

func (this *HeegServer) retry() {
	fmt.Println("Retry create  ", this.Option.Bind())
	transport, err := thrift.NewTServerSocket(this.Option.Bind())
	if nil != err {
		return
	}

	this.transport = transport
	this.server = thrift.NewTSimpleServer4(this.processor, this.transport, this.transportFactory, this.protocolFactory)

	return
}

func (this *HeegServer) Processor(processor thrift.TProcessor) {
	if nil == processor {
		return
	}

	this.processor = processor

	// Debug
	// this.server = thrift.NewTSimpleServer4(processor, this.transport, this.transportFactory, thrift.NewTDebugProtocolFactory(this.protocolFactory, "[Debug]"))
	this.server = thrift.NewTSimpleServer4(processor, this.transport, this.transportFactory, this.protocolFactory)

	return
}

func (this *HeegServer) Run() (err error) {
retry:
	err = this.server.Serve()
	if nil != err {
		if strings.Contains(err.Error(), "address already in use") {
			fmt.Println(this.Option.Bind() + " already in use, 3s retry!")

			time.Sleep(3 * time.Second)

			this.Option.Port += 1
			this.retry()

			goto retry
		}

		return
	}

	return
}
