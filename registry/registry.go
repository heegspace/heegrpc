package registry

import (
	"errors"
	"heegrpc"
	"sync"
	"time"

	"github.com/heegspace/heegproto/s2sname"
)

type Registry struct {
	s2sName map[string][]*S2sName

	mutex sync.Mutex
	watch bool

	client  *s2sname.S2snameServiceClient
	s2shost string // s2s服务的地址信息"主机:端口"
}

var _registry *Registry

func NewRegistry(host string) *Registry {
	if nil != _registry {
		return _registry
	}

	_registry = &Registry{
		watch:   false,
		s2sName: make(map[string][]*S2sName),
	}

	return _registry
}

// 初始化s2sname请求客户端
// 主要用于和s2sname服务进行通信
func InitS2SClient() {
	client := heegrpc.NewHeegRpcClient()
	thclient := example.NewS2snameServiceClientFactory(client.Client())

	this.client = thclient

	return
}

// 更新对应服务的prority
func (this *Registry) IncPrority(s2s *S2sName) {
	if nil == s2s {
		return
	}

	// update to s2sname by thrift

	return
}

// 选择可以用的服务
// @param s2sname
//
func (this *Registry) Selector(s2sname string) (r *S2sName, err error) {
	if 0 == len(s2sname) {
		err = errors.New("s2sname is empty.")

		return
	}

	if _, ok := this.s2sName[s2sname]; !ok {
		err = errors.New("Didn't s2sname's service!")

		return
	}

	index := 0
	prority := int32(999)
	for k, v := range this.s2sName[s2sname] {
		if prority > v.Prority {
			index = k
			prority = v.Prority

			continue
		}
	}

	r = this.s2sName[s2sname][index]
	// 更新s2s prority 到服务器

	return
}

// 通过名称获取对应的s2s列表
//
func (this *Registry) fetchs2sByName(name string) (err error) {
	if "" == name {
		err = errors.New("Name is empty.")

		return
	}

	s2sres, err := this.client.FetchS2sname(name)
	if nil != err || nil == s2sres {
		return
	}

	if 0 != s2sres.Rescode {
		return
	}

	if 0 == len(s2sres.S2ss) {
		return
	}

	for _, v := range s2sres.S2ss {
		if _, ok := this.s2sName[v.Name]; !ok {
			this.s2sName[v.Name] = make([]*S2sName, 0)
		}

	}

	return
}

// 从服务器获取对应的s2s服务信息列表
// 并更新到s2sname数组中
//
func (this *Registry) fetchs2s() (err error) {

	return
}

// 10分钟获取一次s2sname信息
// 并刷新本地列表
func (this *Registry) Watch() {
	if this.watch {
		return
	}

	this.watch = true
	ticker := time.NewTicker(10 * 60 * time.Second)
	for {
		select {
		case <-ticker.C:
			this.fetchs2s()
		}
	}
}
