package rpc

import (
	"fmt"
	"heegrpc/utils"
	"strings"
	"time"

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
	addr, err := utils.ExternalIP()
	if nil != err {
		panic(err)
	}

	v := &HeegServer{
		server: nil,
		inited: false,
		option: Option{
			Addr: addr.String(),
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

retry:
	this.transport, err = thrift.NewTServerSocket(this.option.Bind())
	if nil != err {
		if strings.Contains(err.Error(), "address already in use") {
			fmt.Println(this.option.Bind() + " already in use, 3s retry!")

			time.Sleep(3 * time.Second)

			this.option.Port += 1

			goto retry
		}

		return
	}

	fmt.Println("Service start in " + this.option.Bind())
	this.inited = true
	return
}

func (this *HeegServer) Processor(processor thrift.TProcessor) {
	if nil == processor {
		return
	}

	// Debug
	// this.server = thrift.NewTSimpleServer4(processor, this.transport, this.transportFactory, thrift.NewTDebugProtocolFactory(this.protocolFactory, "[Debug]"))
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
