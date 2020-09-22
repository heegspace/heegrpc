package rpc

import (
	"github.com/apache/thrift/lib/go/thrift"
)

type HeegServer struct {
	server *thrift.TSimpleServer

	transport        thrift.TServerTransport
	protocolFactory  thrift.TProtocolFactory
	transportFactory thrift.TTransportFactory

	option Option

	//
	inited bool
}

func NewHeegServer() *HeegServer {
	v := &HeegServer{
		server: nil,
		inited: false,
		option: Option{
			Addr: "0.0.0.0",
			Port: 8088,
		},
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
	this.transport, err = thrift.NewTServerSocket(this.option.Bind())
	if nil != err {
		return
	}

	this.inited = true
	return
}

func (this *HeegServer) Processor(processor thrift.TProcessor) {
	if nil == processor {
		return
	}

	this.server = thrift.NewTSimpleServer4(processor, this.transport, this.transportFactory, this.protocolFactory)

	return
}

func (this *HeegServer) Run() (err error) {
	err = this.server.Serve()
	if nil != err {
		return
	}

	return
}
