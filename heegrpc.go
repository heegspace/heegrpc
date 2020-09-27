package heegrpc

import (
	"heegrpc/rpc"
)

var _heegServer *rpc.HeegServer
var _heegClient *rpc.HeegClient

// 创建rpc服务器对象
func NewHeegRpcServer() *rpc.HeegServer {
	if nil != _heegServer {
		return _heegServer
	}

	v := rpc.NewHeegServer()

	return v
}

// 创建rpc客户对象
func NewHeegRpcClient(s2sname string) *rpc.HeegClient {
	if nil != _heegClient {
		return _heegClient
	}

	v := rpc.NewHeegClient()

	return v
}
