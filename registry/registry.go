package registry

import (
	"errors"
	"fmt"
	"heegrpc"
	"sync"
	"time"

	"github.com/heegspace/heegproto/s2sname"
)

type Registry struct {
	s2sName map[string][]*S2sName

	mutex sync.Mutex
	watch bool

	client *s2sname.S2snameServiceClient

	S2sname string
	S2shost string // s2s服务的地址信息"主机:端口"
	S2spost int    //
}

var _registry *Registry

// 初始化当前节点信息
//
func NewRegistry() *Registry {
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
// 
func Init(option *heegrpc.Option) (err error) {
	this.S2sname = option.S2sName
	this.S2shost = option.Addr
	this.S2spost = option.Port
	
	client := heegrpc.NewHeegRpcClient()
	thclient := s2sname.NewS2snameServiceClientFactory(client.Client())

	this.client = thclient

	return
}

// 判断是否能够注册
//
func (this *Registry) can() (err error) {
	if 0 == len(this.S2sname) || 0 == len(this.S2shost) {
		err = errors.New("Can't register to registry!")

		return
	}

	err = nil
	return
}

// 注册当前节点到s2s服务
//
func (this *Registry) Register() (err error) {
	err = this.can()
	if nil != err {
		return 
	}

	req := &s2sname.RegisterReq {
		Name: this.S2sname,
		S2s: &&s2sname.S2sname {
			Host: this.S2shost,
			Port: this.S2spost,
			Prority: this.Prority,
			Name: this.S2sname,
		},
	}

	res,err := this.client.RegisterS2sname(req)
	if nil != err {
		return 
	}

	return
}

// 更新对应服务的prority
//
// @param s2s s2s节点
func (this *Registry) IncPrority(s2s *s2sname.S2sname) {
	if nil == s2s {
		return
	}

	// update to s2sname by thrift
	req := &s2sname.UpdateReq{
		Name: s2s.Name,
		S2s: &s2sname.S2sname{
			Host:    s2s.Host,
			Port:    s2s.Port,
			Prority: s2s.Prority,
			Name:    s2s.Name,
		},
	}

	s2sres, err := this.client.UpdateS2sname(req)
	if nil != err {
		return
	}

	return
}

// 选择可以用的服务,选择负载最小的服务器
//
// @param s2sname
// @return r 	s2s节点信息
func (this *Registry) Selector(s2sname string) (r *S2sName, err error) {
	if 0 == len(s2sname) {
		err = errors.New("s2sname is empty.")

		return
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

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
	this.IncPrority(r)

	return
}

// 通过名称获取对应的s2s列表
//
// @param name  s2s名称
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

	this.mutex.Lock()
	defer this.mutex.Unlock()

	for _, v := range s2sres.S2ss {
		if _, ok := this.s2sName[v.Name]; !ok {
			this.s2sName[v.Name] = make([]*S2sName, 0)
		}

		// 更新或插入s2s信息【可优化】
		exists := false
		list := this.s2sName[v.Name]
		for k1, v1 := range list {

			// 找出是否已经存在相同的节点信息
			// 如果存在则直接更新
			vitem := fmt.Sprintf("%s:%d", v.Host, v.Port)
			v1item := fmt.Sprintf("%s:%d", v1.Host, v1.Port)

			if vitem == v1item {
				exists = true
				this.s2sName[v.Name][k1] = v

				break
			}
		}

		// 不存在则将节点追加到管理器中
		if !exists {
			this.s2sName[v.Name] = append(this.s2sName[v.Name], v)
		}
	}

	return
}

// 从服务器获取对应的s2s服务信息列表
// 并更新到s2sname数组中
//
func (this *Registry) fetchs2s() (err error) {
	s2sres, err := this.client.FetchS2snames(name)
	if nil != err || nil == s2sres {
		return
	}

	if 0 != s2sres.Rescode {
		return
	}

	if 0 == len(s2sres.S2ss) {
		return
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	for _, v := range s2sres.S2ss {
		if _, ok := this.s2sName[v.Name]; !ok {
			this.s2sName[v.Name] = make([]*S2sName, 0)
		}

		// 更新或插入s2s信息【可优化】
		exists := false
		list := this.s2sName[v.Name]
		for k1, v1 := range list {

			// 找出是否已经存在相同的节点信息
			// 如果存在则直接更新
			vitem := fmt.Sprintf("%s:%d", v.Host, v.Port)
			v1item := fmt.Sprintf("%s:%d", v1.Host, v1.Port)

			if vitem == v1item {
				exists = true
				this.s2sName[v.Name][k1] = v

				break
			}
		}

		// 不存在则将节点追加到管理器中
		if !exists {
			this.s2sName[v.Name] = append(this.s2sName[v.Name], v)
		}
	}

	return
}

// 20分钟获取一次s2sname信息
// 并刷新本地列表
func (this *Registry) Watch() {
	if this.watch {
		return
	}

	this.watch = true
	ticker := time.NewTicker(20 * 60 * time.Second)
	for {
		select {
		case <-ticker.C:
			this.fetchs2s()
		}
	}
}
