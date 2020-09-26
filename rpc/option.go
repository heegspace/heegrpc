package rpc

import (
	"strconv"
)

type Option struct {
	Addr string
	Port int

	S2sName     string
	S2sKey      string
	CallTimeout int
}

func (this *Option) Bind() string {
	return this.Addr + ":" + strconv.Itoa(this.Port)
}
