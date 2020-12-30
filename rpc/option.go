package rpc

import (
	"strconv"

	"github.com/heegspace/thrift"
)

type Option struct {
	Addr string
	Port int

	Url     string // 主要用于获取s2s地址信息
	S2sName string // 节点的s2sname
	S2sKey  string // 节点的s2s key信息

	CallTimeout int

	ListenFunc thrift.LISTEN_FUNC // 监听成功的回调函数
}

func (this *Option) Bind() string {
	return this.Addr + ":" + strconv.Itoa(this.Port)
}
