// Package proxy is a registry plugin for the micro proxy
package registry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/util/addr"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type SysInfo struct {
	HostName   string   `json:"hostname,omitempty"`
	OS         string   `json:"os,omitempty"`
	CpuNum     int      `json:"cpu_num,omitempty"`
	CpuPercent float64  `json:"cpu_percent,omitempty"`
	MemTotal   uint64   `json:"mem_total,omitempty"`
	MemUsed    uint64   `json:"mem_used,omitempty"`
	DiskTotal  uint64   `json:"disk_total,omitempty"`
	DiskUsed   uint64   `json:"disk_used,omitempty"`
	Ips        []string `json:"ips,omitempty"`
}

func getsysinfo() string {
	var sysinfo SysInfo

	hostinfo, err := host.Info()
	if nil == err {
		sysinfo.HostName = hostinfo.Hostname
		sysinfo.OS = hostinfo.OS
	}

	count, err := cpu.Counts(true)
	if nil == err {
		sysinfo.CpuNum = count
	}

	percent, err := cpu.Percent(time.Second, false)
	if nil == err && 0 < len(percent) {
		sysinfo.CpuPercent = percent[0]
	}

	memInfo, err := mem.VirtualMemory()
	if nil == err && nil != memInfo {
		sysinfo.MemTotal = memInfo.Total
		sysinfo.MemUsed = memInfo.Used
	}

	parts, err := disk.Partitions(true)
	if nil == err && 0 < len(parts) {
		for _, v := range parts {
			diskInfo, err := disk.Usage(v.Mountpoint)
			if nil != err {
				continue
			}

			sysinfo.DiskTotal = sysinfo.DiskTotal + diskInfo.Total
			sysinfo.DiskUsed = sysinfo.DiskUsed + diskInfo.Used
		}

	}

	sysinfo.Ips = addr.IPs()
	info, _ := json.Marshal(sysinfo)
	return string(info)
}

type proxy struct {
	opts registry.Options
}

func init() {
	cmd.DefaultRegistries["proxy"] = NewRegistry
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

func newRegistry(opts ...registry.Option) registry.Registry {
	s := &proxy{
		opts: registry.Options{},
	}
	configure(s, opts...)
	return s
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

	if nil == service.Metadata {
		service.Metadata = make(map[string]string)
	}
	service.Metadata["sysinfo"] = getsysinfo()

	b, err := json.Marshal(service)
	if err != nil {
		return err
	}

	var gerr error
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
		return nil
	}
	return gerr
}

func (s *proxy) Deregister(service *registry.Service, opts ...registry.DeregisterOption) error {
	b, err := json.Marshal(service)
	if err != nil {
		return err
	}

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
		return nil
	}
	return gerr
}

func (s *proxy) GetService(service string, opts ...registry.GetOption) ([]*registry.Service, error) {
	if 0 == len(service) {
		logger.Info("service", service)
	}
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
		var services []*registry.Service
		if err := json.Unmarshal(b, &services); err != nil {
			gerr = err
			continue
		}

		data, _ := json.Marshal(services)
		logger.Info("GetService ", service, string(data))

		return services, nil
	}

	return nil, gerr
}

func (s *proxy) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	var gerr error
	logger.Info("ListServices")
	return nil, gerr
}

func (s *proxy) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	logger.Info("Watch--------")
	var gerr error
	return nil, gerr
}

func (s *proxy) String() string {
	return "proxy"
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	return newRegistry(opts...)
}
