package rpc

import (
	"github.com/heegspace/thrift"
)

type HeegClient struct {
	transport        thrift.TTransport
	protocolFactory  thrift.TProtocolFactory
	transportFactory thrift.TTransportFactory

	//
	inited bool

	option Option
}

func NewHeegClient(option Option) *HeegClient {
	v := &HeegClient{
		inited: false,
		option: option,
	}

	return v
}

func (this *HeegClient) Init() (err error) {
	if this.inited {
		return
	}

	this.protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	this.transportFactory = thrift.NewTBufferedTransportFactory(8192)
	this.transportFactory = thrift.NewTFramedTransportFactory(this.transportFactory)

	this.inited = true
	return
}

func (this *HeegClient) Close() {
	this.transport.Close()

	return
}

// 获取client对象
//
// @return 返回用于创建thrift client的信息
func (this *HeegClient) Client() (thrift.TTransport, thrift.TProtocolFactory) {
	if !this.inited {
		this.Init()
	}

	// 主要是获取thrift服务的地址信息
	tt, err := thrift.NewTSocket(this.option.Bind())
	if nil != err {
		panic(err.Error())
	}

	this.transport, err = this.transportFactory.GetTransport(tt)
	if err != nil {
		panic(err.Error())
	}
	if err := this.transport.Open(); err != nil {
		panic(err.Error())
	}

	// Debug
	// return this.transport, thrift.NewTDebugProtocolFactory(this.protocolFactory, "Client")
	return this.transport, this.protocolFactory
}
