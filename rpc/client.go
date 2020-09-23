package rpc

import (
	"github.com/apache/thrift/lib/go/thrift"
)

type HeegClient struct {
	transport        thrift.TTransport
	protocolFactory  thrift.TProtocolFactory
	transportFactory thrift.TTransportFactory

	option Option

	//
	inited bool
}

func NewHeegClient() *HeegClient {
	v := &HeegClient{
		inited: false,
		option: Option{
			Addr: "0.0.0.0",
			Port: 8088,
		},
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

// 通过s2sname获取client对象
//
// @param s2sname 	s2sname名字
// @return 返回用于创建thrift client的信息
func (this *HeegClient) Client(s2sname string) (thrift.TTransport, thrift.TProtocolFactory) {
	if !this.inited {
		this.Init()
	}

	// get server addr by s2sname
	// 主要是获取thrift服务的地址信息
	tt, err := thrift.NewTSocket(this.option.Bind())
	this.transport, err = this.transportFactory.GetTransport(tt)
	if err != nil {
		panic(err.Error())
	}
	if err := this.transport.Open(); err != nil {
		panic(err.Error())
	}

	return this.transport, this.protocolFactory
}
