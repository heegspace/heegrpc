package rpc

import (
	"strconv"
)

type Option struct {
	Addr string
	Port int
}

func (this *Option) Bind() string {
	return this.Addr + ":" + strconv.Itoa(this.Port)
}
