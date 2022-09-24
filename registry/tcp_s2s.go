package registry

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/heegspace/appcom"
	"github.com/heegspace/heegapo"
	"go-micro.dev/v4/logger"
	"go.uber.org/zap"
)

type StreamReq struct {
	Cmd   string            `json:"cmd"`
	Data  string            `json:"data"`
	Tag   string            `json:"tag"`
	Extra map[string]string `json:"extra"`
}

type StreamRes struct {
	Cmd  string `json:"cmd"`
	Code string `json:"code"`
	Data string `json:"data"`
	Tag  string `json:"tag"`
}

type tcpS2s struct {
	conn   *net.TCPConn
	rwlock sync.RWMutex

	addr string
}

var once sync.Once
var g_s2sCli *tcpS2s

func TcpS2s() *tcpS2s {
	once.Do(func() {
		addr := ""
		ip := heegapo.DefaultApollo.Config("heegspace.common.yaml", "s2s", "tcp_ip").String("")
		port := heegapo.DefaultApollo.Config("heegspace.common.yaml", "s2s", "tcp_port").Int64(-1)
		if len(ip) != 0 && 0 < port {
			addr = fmt.Sprintf("%s:%d", ip, port)
		}
		if nil == g_s2sCli {
			g_s2sCli = &tcpS2s{
				rwlock: sync.RWMutex{},
				addr:   addr,
			}
		}
	})

	return g_s2sCli
}

func (this *tcpS2s) enable() bool {
	if len(this.addr) != 0 {
		return true
	}

	return false
}

func (this *tcpS2s) GetConn() *net.TCPConn {
	this.rwlock.RLock()
	defer this.rwlock.RUnlock()

	return this.conn
}

func (this *tcpS2s) reset() {
	this.rwlock.Lock()
	this.conn = nil
	this.rwlock.Unlock()
}

func (this *tcpS2s) Connect() {
	if len(this.addr) == 0 {
		return
	}

	for {
		tcpAddr, err := net.ResolveTCPAddr("tcp4", this.addr)
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			logger.Error("Connect to s2s fail, 2s reconnected!", zap.Error(err))

			time.Sleep(2 * time.Second)
			continue
		}

		this.rwlock.Lock()
		this.conn = conn
		this.rwlock.Unlock()
		break
	}

	logger.Info("Connect to s2s success!")
}

func (s *proxy) onStart() {
	if !TcpS2s().enable() {
		return
	}

	go func() {
		i := 0
		for {
			appcom.ReadFromTcp(TcpS2s().GetConn(), func(ctx context.Context, conn *net.TCPConn, size int, data []byte) (err error) {
				var res StreamRes
				buf := bytes.NewBufferString(string(data))
				dec := gob.NewDecoder(buf)
				err = dec.Decode(&res)
				if nil != err {
					logger.Error("ReadFromTcp err", zap.Error(err))

					return
				}

				logger.Debug("ReadFromTcp start", zap.Any("size", size), zap.Any("cmd", res.Cmd), zap.Any("code", res.Code), zap.Any("tag", res.Tag))
				if "notify" != res.Cmd {
					switch res.Cmd {
					case "update":

					case "delete":

					case "get":
						s.chlock.RLock()
						if _, ok := s.upch[res.Tag]; ok {
							s.upch[res.Tag] <- res.Data
						}
						s.chlock.RUnlock()

					case "gets":
						s.chlock.RLock()
						if _, ok := s.upch[res.Tag]; ok {
							s.upch[res.Tag] <- res.Data
						}
						s.chlock.RUnlock()
					}

					return
				}

				// s2s服务主动推送的消息
				// 收到通知重新获取节点信息
				switch res.Code {
				case "update":
					s.refresh <- true
				case "delete":
					s.refresh <- true
				}

				logger.Debug("ReadFromTcp refresh", zap.Any("size", size), zap.Any("cmd", res.Cmd), zap.Any("code", res.Code))
				return nil
			}, func(ctx context.Context, conn *net.TCPConn) error {
				logger.Warn("s2s connected closed! start retry!")
				TcpS2s().reset()

				return nil
			})

			i++
			logger.Debug("ReadFromTcp start reconnect!", zap.Any("times", i))

			time.Sleep(2 * time.Second)
			TcpS2s().Connect()
		}

		return
	}()
}
