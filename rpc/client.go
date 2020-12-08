package rpc

import (
	"time"

	"github.com/heegspace/thrift"
	log "github.com/sirupsen/logrus"
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
//
func (this *HeegClient) Client() (*thrift.TStandardClient, *thrift.TBufferedTransport) {
	if !this.inited {
		this.Init()
	}

retry:
	// 主要是获取thrift服务的地址信息
	tt, err := thrift.NewTSocket(this.option.Bind())
	if nil != err {
		log.Println(this.option, "NewTSocket err ", err, "  2s retry.")

		time.Sleep(2 * time.Second)
		goto retry
	}
	trans := thrift.NewTBufferedTransport(tt, 1*1024*1024)

	if err = tt.Open(); err != nil {
		log.Println(this.option, "NewTBufferedTransport Open() err ", err, "  2s retry.")

		time.Sleep(2 * time.Second)
		goto retry
	}

	iprot := this.protocolFactory.GetProtocol(trans)
	oprot := this.protocolFactory.GetProtocol(trans)

	return thrift.NewTStandardClient(iprot, oprot), trans
}
