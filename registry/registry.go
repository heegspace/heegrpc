package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/heegspace/heegproto/s2sname"
	"github.com/heegspace/heegrpc"
	"github.com/heegspace/heegrpc/rpc"
)

var defaultCtx = context.Background()

type Registry struct {
	s2sName map[string][]*S2sName // s2snode信息管理结构

	mutex sync.Mutex
	watch bool

	S2sname string // 当前节点的s2sname
	S2shost string // 当前节点的监听地址
	S2spost int    // 当前节点的监听端口号

	RegConf *registry_conf

	client *rpc.HeegClient
}

// registry管理对象
// 每个节点仅有一个registry管理对象
// 主要是用于管理s2s相关的信息
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

// 通过发起http请求，从http服务器中获取s2s信息
// 主要用于连接s2s服务器，用于注册和发现服务
//
func (this *Registry) s2sInfo(url string) {
retry:
	resp, err := http.Get(url)
	if err != nil {
		time.Sleep(2 * time.Second)
		goto retry
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		log.Println("s2sinfo ReadAll err ", err)

		time.Sleep(2 * time.Second)
		goto retry
	}

	if 200 != resp.StatusCode {
		log.Println("S2s info Request fail, ", string(body))

		time.Sleep(2 * time.Second)
		goto retry
	}

	var register registry_conf
	err = json.Unmarshal(body, &register)
	if nil != err {
		log.Println("s2sinfo Unmarshal err ", err)

		time.Sleep(2 * time.Second)
		goto retry
	}

	this.RegConf = &register
	return
}

// 初始化s2sname请求客户端
// 主要用于和s2sname服务进行通信
//
// @param option
//
func (this *Registry) Init(option *rpc.Option) (err error) {
	this.s2sInfo(option.Url)

	this.S2sname = option.S2sName
	this.S2shost = option.Addr
	this.S2spost = option.Port

	var _optn rpc.Option
	_optn.Addr = this.RegConf.Host
	_optn.Port = this.RegConf.Port

	this.client = heegrpc.NewHeegRpcClient(_optn)

	this.fetchs2s()

	// 启动后台任务
	go this.Watch()

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
// 将当前节点注册到s2s服务中，以便其他服务发现这个服务
//
func (this *Registry) Register() (err error) {
	err = this.can()
	if nil != err {
		return
	}

	req := &s2sname.RegisterReq{
		Name: this.S2sname,
		S2s: &s2sname.S2sname{
			Host:    this.S2shost,
			Port:    int32(this.S2spost),
			Prority: 0,
			Name:    this.S2sname,
		},
	}

	thclient := s2sname.NewS2snameServiceClient(this.client.Client())
	res, err := thclient.RegisterS2sname(defaultCtx, req)
	if nil != err {
		return
	}

	go this.Heart()

	fmt.Println("RegisterS2sname: ", res)
	return
}

// 更新对应服务的prority
//
// @param s2s s2s节点
//
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

	thclient := s2sname.NewS2snameServiceClient(this.client.Client())
	s2sres, err := thclient.UpdateS2sname(defaultCtx, req)
	if nil != err {
		return
	}
	fmt.Println("UpdateS2sname: ", s2sres)

	return
}

// 选择服务,选择负载最小的服务器
//
// @param name
// @return r 	s2s节点信息
//
func (this *Registry) Selector(name string) (r *S2sName, err error) {
	if 0 == len(name) {
		err = errors.New("Selector name is empty.")

		return
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	if _, ok := this.s2sName[name]; !ok {
		err = errors.New("Didn't name's service!")

		return
	}

	index := 0
	prority := int32(999999999)
	for k, v := range this.s2sName[name] {
		if prority > v.Prority {
			index = k
			prority = v.Prority

			continue
		}
	}

	r = &S2sName{
		Host:    this.s2sName[name][index].Host,
		Port:    this.s2sName[name][index].Port,
		Prority: this.s2sName[name][index].Prority,
	}

	value := &s2sname.S2sname{
		Host:    r.Host,
		Port:    r.Port,
		Name:    name,
		Prority: prority + 1,
	}

	// 更新s2s prority 到服务器
	this.IncPrority(value)

	return
}

// 通过名称获取对应的s2s列表
//
// @param name  s2s名称
//
func (this *Registry) fetchs2sByName(name string) (err error) {
	if "" == name {
		err = errors.New("Name is empty.")

		return
	}

	thclient := s2sname.NewS2snameServiceClient(this.client.Client())
	s2sres, err := thclient.FetchS2sname(defaultCtx, name)
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
				this.s2sName[v.Name][k1] = v1

				break
			}
		}

		// 不存在则将节点追加到管理器中
		if !exists {
			value := &S2sName{
				Host:    v.Host,
				Port:    v.Port,
				Prority: v.Prority,
			}

			this.s2sName[v.Name] = append(this.s2sName[v.Name], value)
		}
	}

	return
}

// 从服务器获取对应的s2s服务信息列表
// 并更新到s2sname数组中
//
func (this *Registry) fetchs2s() (err error) {
	thclient := s2sname.NewS2snameServiceClient(this.client.Client())
	s2sres, err := thclient.FetchS2snames(defaultCtx)
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

				value := &S2sName{
					Host:    v.Host,
					Port:    v.Port,
					Prority: v.Prority,
				}

				this.s2sName[v.Name][k1] = value

				break
			}
		}

		// 不存在则将节点追加到管理器中
		if !exists {
			value := &S2sName{
				Host:    v.Host,
				Port:    v.Port,
				Prority: v.Prority,
			}

			this.s2sName[v.Name] = append(this.s2sName[v.Name], value)
		}
	}

	// data, _ := json.Marshal(this.s2sName)
	// fmt.Println("fetchs2s: ", string(data))
	return
}

func (this *Registry) heart() {
	err := this.can()
	if nil != err {
		fmt.Println("heart can error ", err)

		return
	}

	req := &s2sname.HeartReq{
		Name: this.S2sname,
		S2s: &s2sname.S2sname{
			Host:    this.S2shost,
			Port:    int32(this.S2spost),
			Prority: 0,
			Name:    this.S2sname,
		},
	}

	thclient := s2sname.NewS2snameServiceClient(this.client.Client())
	_, err = thclient.Heart(defaultCtx, req)
	if nil != err {
		fmt.Println("Send Heart error ", err)
		return
	}

	return
}

// 维护s2s连接的心跳包
//
func (this *Registry) Heart() {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			this.heart()
		}
	}
}

// 定时获取一次s2sname信息
// 并刷新本地列表
//
func (this *Registry) Watch() {
	if this.watch {
		return
	}
	this.watch = true

	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			err := this.fetchs2s()
			if nil != err {
				fmt.Println("Watch fetchs2s err ", err)
			}
		}
	}
}
