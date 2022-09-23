// Package proxy is a registry plugin for the micro proxy
package registry

import (
	"crypto/rand"
	"encoding/json"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"go-micro.dev/v4/util/addr"
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

func getRandomTag() string {
	alphanum := "0123456789"
	var bytes = make([]byte, 32)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}
