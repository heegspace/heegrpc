package heegrpc

import (
	"github.com/heegspace/heegrpc/rpc"
)

var _heegServer *rpc.HeegServer
var _heegClient *rpc.HeegClient

// 创建rpc服务器对象
func NewHeegRpcServer(option rpc.Option) *rpc.HeegServer {
	if nil != _heegServer {
		return _heegServer
	}

	v := rpc.NewHeegServer(option)

	return v
}

// 创建rpc客户对象
func NewHeegRpcClient(option rpc.Option) *rpc.HeegClient {
	if nil != _heegClient {
		return _heegClient
	}

	v := rpc.NewHeegClient(option)

	return v
}
