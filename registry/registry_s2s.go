// Package proxy is a registry plugin for the micro proxy
package registry

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/heegspace/appcom"
	"go-micro.dev/v4/cmd"
	"go-micro.dev/v4/logger"
	"go-micro.dev/v4/registry"
)

type proxy struct {
	opts registry.Options

	rwlock sync.RWMutex
	svrs   map[string][]*registry.Service

	refresh chan bool
	upch    map[string]chan string
	chlock  sync.RWMutex
}

var watchNode []string

func init() {
	watchNode = make([]string, 0)
	cmd.DefaultRegistries["proxy"] = NewRegistry
}

// 设置要监听的节点连接信息
//
// @param nodes 配置中的nodes项
//
func SetWatchNode(nodes []string) {
	if 0 == len(nodes) {
		return
	}

	for _, v := range nodes {
		watchNode = append(watchNode, v)
	}

	return
}

func configure(s *proxy, opts ...registry.Option) error {
	for _, o := range opts {
		o(&s.opts)
	}
	var addrs []string
	for _, addr := range s.opts.Addrs {
		if len(addr) > 0 {
			addrs = append(addrs, addr)
		}
	}
	if len(addrs) == 0 {
		addrs = []string{"localhost:8081"}
	}

	registry.Addrs(addrs...)(&s.opts)
	return nil
}

var gs *proxy

func newRegistry(opts ...registry.Option) registry.Registry {
	if nil == gs {
		gs = &proxy{
			opts:    registry.Options{},
			rwlock:  sync.RWMutex{},
			svrs:    make(map[string][]*registry.Service),
			upch:    make(map[string]chan string),
			refresh: make(chan bool, 2),
			lock:    sync.RWMutex{},
		}

		go gs.refresh()
		configure(gs, opts...)
	}

	return gs
}

func (s *proxy) Init(opts ...registry.Option) error {
	return configure(s, opts...)
}

func (s *proxy) Options() registry.Options {
	return s.opts
}

func (s *proxy) Register(service *registry.Service, opts ...registry.RegisterOption) error {
	if nil == service {
		err := errors.New("service is nil")

		return err
	}

	if GetDeregister().IsDe() {
		return nil
	}

	if nil == service.Metadata {
		service.Metadata = make(map[string]string)
	}
	service.Metadata["sysinfo"] = getsysinfo()

	b, err := json.Marshal(service)
	if err != nil {
		return err
	}

	var gerr error
	if TcpS2s().enable() {
		var req StreamReq
		req.Cmd = "update"
		req.Data = string(b)
		req.Tag = getRandomTag()
		buf := &bytes.Buffer{}
		enc := gob.NewEncoder(buf)
		err := enc.Encode(req)
		if nil != err {
			return err
		}

		_, err = appcom.WriteToConnections(TcpS2s().GetConn(), buf.Bytes())
		if nil != err {
			return err
		}

		GetDeregister().LocalSvr = service
		return nil
	}

	for _, addr := range s.opts.Addrs {
		scheme := "http"
		if s.opts.Secure {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s/registry", scheme, addr)
		rsp, err := http.Post(url, "application/json", bytes.NewReader(b))
		if err != nil {
			gerr = err
			continue
		}
		if rsp.StatusCode != 200 {
			b, err := ioutil.ReadAll(rsp.Body)
			if err != nil {
				return err
			}
			rsp.Body.Close()
			gerr = errors.New(string(b))
			continue
		}
		io.Copy(ioutil.Discard, rsp.Body)
		rsp.Body.Close()

		GetDeregister().LocalSvr = service
		return nil
	}

	return gerr
}

func (s *proxy) Deregister(service *registry.Service, opts ...registry.DeregisterOption) error {
	b, err := json.Marshal(service)
	if err != nil {
		return err
	}

	// tcp
	if TcpS2s().enable() {
		var req StreamReq
		req.Cmd = "delete"
		req.Data = string(b)
		req.Tag = getRandomTag()
		buf := &bytes.Buffer{}
		enc := gob.NewEncoder(buf)
		err := enc.Encode(req)
		if nil != err {
			return err
		}

		_, err = appcom.WriteToConnections(TcpS2s().GetConn(), buf.Bytes())
		if nil != err {
			return err
		}

		GetDeregister().De()
		return nil
	}

	// http
	var gerr error
	for _, addr := range s.opts.Addrs {
		scheme := "http"
		if s.opts.Secure {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s/registry", scheme, addr)

		req, err := http.NewRequest("DELETE", url, bytes.NewReader(b))
		if err != nil {
			gerr = err
			continue
		}

		rsp, err := http.DefaultClient.Do(req)
		if err != nil {
			gerr = err
			continue
		}

		if rsp.StatusCode != 200 {
			b, err := ioutil.ReadAll(rsp.Body)
			if err != nil {
				return err
			}
			rsp.Body.Close()
			gerr = errors.New(string(b))
			continue
		}

		io.Copy(ioutil.Discard, rsp.Body)
		rsp.Body.Close()

		GetDeregister().De()

		return nil
	}

	return gerr
}

// 优先读取内存中的服务信息
//
// @param service 	服务名
// @return {[]Service,error}
//
func (s *proxy) GetService(service string, opts ...registry.GetOption) ([]*registry.Service, error) {
	if 0 == len(service) {
		logger.Info("service", service)

		return nil, errors.New("Service name is nil")
	}

	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	if _, ok := s.svrs[service]; ok {
		item := make([]*registry.Service, 0)
		for _, v := range s.svrs[service] {
			var svr registry.Service

			svr = *v
			item = append(item, &svr)
		}

		return item, nil
	}

	logger.Info(service, " node cache not exists.")
	return s.getService(service)
}

func (s *proxy) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	var gerr error
	logger.Info("ListServices")

	return nil, gerr
}

func (s *proxy) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}
	logger.Info("Watch, Service: ", wo.Service)

	return newWatcher("")
}

func (s *proxy) String() string {
	return "proxy"
}

// 根据服务名获取服务列表
//
// @param service 	服务名
// @return {[]Service}
//
func (s *proxy) getService(service string) ([]*registry.Service, error) {
	if 0 == len(service) {
		return nil, errors.New("Service name is nil")
	}

	// tcp
	var services []*registry.Service
	if TcpS2s().enable() {
		var req StreamReq
		req.Cmd = "get"
		req.Data = service
		req.Tag = getRandomTag()
		buf := &bytes.Buffer{}
		enc := gob.NewEncoder(buf)
		err := enc.Encode(req)
		if nil != err {
			return nil, err
		}

		_, err = appcom.WriteToConnections(TcpS2s().GetConn(), buf.Bytes())
		if nil != err {
			return nil, err
		}

		// wait response, timeout 300ms
		result := ""
		this.chlock.Lock()
		this.upch[req.Tag] = make(chan string, 1)
		this.chlock.Unlock()
		defer func() {
			this.chlock.Lock()
			close(this.upch[req.Tag])
			delete(this.upch, req.Tag)
			this.chlock.Unlock()
		}()
		select {
		case msg, ok := <-this.upch[req.Tag]:
			if ok {
				result = msg
			}
		case <-time.After(time.Millisecond * time.Duration(300)):
			logger.Debug("getService wait response timeout!", zap.Any("s2sname", service))

			return nil, errors.New("getService " + service + " timeout!")
		}

		if len(result) == 0 {
			logger.Debug("getService wait response return empty!")

			return nil, errors.New("Didn't node info")
		}

		if err := json.Unmarshal([]byte(result), &services); err != nil {
			logger.Debug("getService Unmarshal err!", zap.Any("result", result), zap.Error(err))

			return nil, err
		}

		return services, nil
	}

	// http
	var gerr error
	for _, addr := range s.opts.Addrs {
		scheme := "http"
		if s.opts.Secure {
			scheme = "https"
		}

		url := fmt.Sprintf("%s://%s/registry/%s", scheme, addr, url.QueryEscape(service))
		rsp, err := http.Get(url)
		if err != nil {
			gerr = err
			continue
		}

		if rsp.StatusCode != 200 {
			b, err := ioutil.ReadAll(rsp.Body)
			if err != nil {
				return nil, err
			}
			rsp.Body.Close()
			gerr = errors.New(string(b))
			continue
		}

		b, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			gerr = err
			continue
		}
		rsp.Body.Close()

		if err := json.Unmarshal(b, &services); err != nil {
			gerr = err
			continue
		}

		return services, nil
	}

	return nil, gerr
}

// 根据服务名批量获取服务列表
//
// @param s2sname 	服务名
// @return {[]Service}
//
func (s *proxy) getServices(s2sname string) (map[string][]*registry.Service, error) {
	if 0 == len(s2sname) {
		return nil, errors.New("Service name is nil")
	}

	// tcp
	var services map[string][]*registry.Service
	services = make(map[string][]*registry.Service)
	if TcpS2s().enable() {
		var req StreamReq
		req.Cmd = "gets"
		req.Data = s2sname
		req.Tag = getRandomTag()
		buf := &bytes.Buffer{}
		enc := gob.NewEncoder(buf)
		err := enc.Encode(req)
		if nil != err {
			return nil, err
		}

		_, err = appcom.WriteToConnections(TcpS2s().GetConn(), buf.Bytes())
		if nil != err {
			return nil, err
		}

		// wait response, timeout 300ms
		result := ""
		this.chlock.Lock()
		this.upch[req.Tag] = make(chan string, 1)
		this.chlock.Unlock()
		defer func() {
			this.chlock.Lock()
			close(this.upch[req.Tag])
			delete(this.upch, req.Tag)
			this.chlock.Unlock()
		}()
		select {
		case msg, ok := <-this.upch[req.Tag]:
			if ok {
				result = msg
			}
		case <-time.After(time.Millisecond * time.Duration(300)):
			logger.Debug("getService wait response timeout!", zap.Any("s2sname", service))

			return nil, errors.New("getServices " + service + " timeout!")
		}

		if len(result) == 0 {
			logger.Debug("getService wait response return empty!")

			return nil, errors.New("getServices Didn't node info")
		}

		svrs := strings.Split(s2sname, ",")
		if 1 == len(svrs) {
			var serv []*registry.Service
			if err = json.Unmarshal([]byte(result), &serv); err != nil {
				logger.Debug("getServices Unmarshal err!", zap.Any("result", result), zap.Error(err))

				return nil, err
			}

			services[s2sname] = serv
		} else {
			if err = json.Unmarshal([]byte(result), &services); err != nil {
				logger.Debug("getServices Unmarshal err!", zap.Any("result", result), zap.Error(err))

				return nil, err
			}
		}

		return services, nil
	}

	// http
	var gerr error
	for _, addr := range s.opts.Addrs {
		scheme := "http"
		if s.opts.Secure {
			scheme = "https"
		}

		url := fmt.Sprintf("%s://%s/registry/%s", scheme, addr, url.QueryEscape(s2sname))
		rsp, err := http.Get(url)
		if err != nil {
			gerr = err
			continue
		}

		if rsp.StatusCode != 200 {
			b, err := ioutil.ReadAll(rsp.Body)
			if err != nil {
				return nil, err
			}
			rsp.Body.Close()
			gerr = errors.New(string(b))
			continue
		}

		b, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			gerr = err
			continue
		}
		rsp.Body.Close()

		svrs := strings.Split(s2sname, ",")
		if 1 == len(svrs) {
			var serv []*registry.Service
			if err := json.Unmarshal(b, &serv); err != nil {
				gerr = err
				continue
			}

			services[s2sname] = serv
		} else {
			if err := json.Unmarshal(b, &services); err != nil {
				gerr = err
				continue
			}
		}

		return services, nil
	}

	return nil, gerr
}

// 刷新订阅的s2s信息
//
func (s *proxy) refresh() {
	if 0 == len(watchNode) {
		return
	}

	fn := func() {
		names := strings.Join(watchNode, ",")
		svrs, err := s.getServices(names)
		if nil != err {
			logger.Error("Refresh getService err ", err)

			continue
		}
		for k, v := range svrs {
			if 0 == len(v) {
				continue
			}

			s.rwlock.Lock()
			s.svrs[k] = v
			s.rwlock.Unlock()
		}
	}

	// 10s定时刷新订阅的服务信息
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			fn()
		case <-refresh:
			fn()
		}
	}
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	return newRegistry(opts...)
}
