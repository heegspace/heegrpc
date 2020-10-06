package rpc

import (
	"strconv"
)

type Option struct {
	Addr string
	Port int

	Url     string // 主要用于获取s2s地址信息
	S2sName string
	S2sKey  string

	CallTimeout int
}

func (this *Option) Bind() string {
	return this.Addr + ":" + strconv.Itoa(this.Port)
}
