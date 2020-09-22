package rpc

type HeegClient struct {
}

func NewHeegClient() *HeegClient {
	v := &HeegClient{}

	return v
}

func (this *HeegClient) Init() (err error) {
	return
}

// func (this *HeegClient) Client() (thrift.TTransport, thrift.TProtocolFactory) {
// 	transport, err := thrift.NewTSocket("localhost:9990")

// 	transport, err = transportFactory.GetTransport(transport)
// 	if err != nil {
// 		panic(err.Error())
// 	}
// 	if err := transport.Open(); err != nil {
// 		panic(err.Error())
// 	}

// 	return transport, this.protocolFactory
// }
