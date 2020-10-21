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
func (this *HeegClient) Client() *thrift.TStandardClient {
	if !this.inited {
		this.Init()
	}

	// 主要是获取thrift服务的地址信息
	tt, err := thrift.NewTSocket(this.option.Bind())
	if nil != err {
		panic(err.Error())
	}
	trans := thrift.NewTBufferedTransport(tt, 1*1024*1024)

	if err = tt.Open(); err != nil {
		panic(err.Error())
	}

	iprot := this.protocolFactory.GetProtocol(trans)
	oprot := this.protocolFactory.GetProtocol(trans)

	return thrift.NewTStandardClient(iprot, oprot)
}
