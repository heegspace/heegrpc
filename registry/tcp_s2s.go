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
	Cmd  string `json:"cmd"`
	Data string `json:"data"`
}

type StreamRes struct {
	Cmd  string `json:"cmd"`
	Code string `json:"rescode"`
	Data string `json:"data"`
}

type s2sClient struct {
	conn   *net.TCPConn
	rwlock sync.RWMutex

	addr string
}

var once sync.Once
var g_s2sCli *s2sClient

func S2sClient() *s2sClient {
	once.Do(func() {
		addr := ""
		ip := heegapo.DefaultApollo.Config("heegspace.common.yaml", "s2s", "tcp_ip").String("")
		port := heegapo.DefaultApollo.Config("heegspace.common.yaml", "s2s", "tcp_port").Int32(-1)
		if len(ip) != 0 && 0 < port {
			addr = fmt.Sprintf("%s:%d", ip, port)
		}
		if nil == g_s2sCli {
			g_s2sCli = &s2sClient{
				rwlock: sync.RWMutex{},
				addr:   addr,
			}
		}
	})

	return g_s2sCli
}

func (this *s2sClient) enable() bool {
	if len(this.addr) != 0 {
		return true
	}

	return false
}

func (this *s2sClient) GetConn() *net.TCPConn {
	this.rwlock.RLock()
	defer this.rwlock.RUnlock()

	return this.conn
}

func (this *s2sClient) Connect() {
	if len(this.addr) == 0 {
		return
	}

	for {
		tcpAddr, err := net.ResolveTCPAddr("tcp4", this.addr)
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			logger.Warn("Connect to s2s fail, 2s reconnected!")

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

func (this *s2sClient) onStart() {
	if !this.enable() {
		return
	}

	go func() {
		i := 0
		for {
			appcom.ReadFromTcp(S2sClient().GetConn(), func(ctx context.Context, conn *net.TCPConn, size int, data []byte) (err error) {
				var res StreamRes
				buf := bytes.NewBufferString(string(data))
				dec := gob.NewDecoder(buf)
				err = dec.Decode(&res)
				if nil != err {
					logger.Error("ReadFromTcp err", zap.Error(err))

					return
				}

				logger.Info("ReadFromTcp ", zap.Any("size", size), zap.Any("res", res))
				if "notify" != res.Cmd {
					switch res.Cmd {
					case "update":

					case "delete":

					case "get":

					case "gets":

					}

					return
				}

				// s2s服务主动推送的消息
				switch res.Code {
				case "update":

				case "delete":

				}
				return nil
			}, func(ctx context.Context, conn *net.TCPConn) error {
				logger.Info("CloseCb ----------- ")

				return nil
			})

			i++
			logger.Info("ReadFromTcp start reconnect!", zap.Any("times", i))
			time.Sleep(2 * time.Second)
			S2sClient().Connect()
		}

		return
	}()
}
