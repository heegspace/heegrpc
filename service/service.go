package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	hystrixsrc "github.com/afex/hystrix-go/hystrix"
	"github.com/gin-gonic/gin"

	"github.com/asim/go-micro/plugins/wrapper/breaker/hystrix/v3"
	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/metadata"
	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/server"
	"github.com/juju/ratelimit"
	"github.com/micro/go-micro/v2/config"

	httpClient "github.com/asim/go-micro/plugins/client/http/v3"
	httpServer "github.com/asim/go-micro/plugins/server/http/v3"
	grpc "github.com/asim/go-micro/plugins/transport/grpc/v3"
	ratelimiter "github.com/asim/go-micro/plugins/wrapper/ratelimiter/ratelimit/v3"
	registry "github.com/asim/go-micro/v3/registry"
	foot "github.com/heegspace/heegrpc/callfoot"
	s2s "github.com/heegspace/heegrpc/registry"
)

func errstr(err error) string {
	if nil == err {
		return ""
	}

	return err.Error()
}

type response struct {
	rescode interface{}
	resmsg  interface{}
	extra   interface{}
}

func (obj response) String() string {
	str := fmt.Sprintf("{rescode: %v, resmsg: %v, extra: %v}", obj.rescode, obj.resmsg, obj.extra)

	return str
}

// 客户端调用追踪
func metricsWrap(cf client.CallFunc) client.CallFunc {
	return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
		t := time.Now()
		err := cf(ctx, node, req, rsp, opts)
		md, _ := metadata.FromContext(ctx)
		freq := &foot.RPCFootReq{
			Svrname: req.Service(),
			Method:  req.Method(),
			Remote:  md["Remote"],
			Localip: md["Local"],
			Timeout: int64(time.Since(t)),
			Extra: map[string]string{
				"error": errstr(err),
				"type":  "client",
			},
		}

		var res response
		obj := reflect.ValueOf(rsp)
		elem := obj.Elem()
		if nil == err && nil != rsp && elem.Kind() == reflect.Struct {
			rescode := elem.FieldByName("Rescode")
			if rescode.Kind() == reflect.Interface {
				res.rescode = rescode.Interface()
			}
			resmsg := elem.FieldByName("Resmsg")
			if resmsg.Kind() == reflect.String || resmsg.Kind() == reflect.Interface {
				res.resmsg = resmsg.Interface()
			}

			extra := elem.FieldByName("Extra")
			if extra.Kind() == reflect.Interface {
				res.extra = extra.Interface()
			}
		}

		// 上报数据到统计服务
		var fres foot.RPCFootRes
		ferr := HttpRequest(config.Get("statis", "svrname").String("footnode"), config.Get("static", "rpcmethod").String("/foot/rpc"), freq, &fres, "application/proto")
		logger.Infof("[Metrics Wrapper]-%v, Req: %v, Res: %s ,err: %v, footerr: %v, duration: %v\n", req.Method(), req.Body(), res, err, ferr, time.Since(t))
		return err
	}
}

// 服务端日志跟踪
func logWrapper(fn server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp interface{}) error {
		t := time.Now()
		err := fn(ctx, req, rsp)

		md, _ := metadata.FromContext(ctx)
		freq := &foot.RPCFootReq{
			Svrname: req.Service(),
			Method:  req.Method(),
			Remote:  md["Remote"],
			Localip: md["Local"],
			Timeout: int64(time.Since(t)),
			Extra: map[string]string{
				"error": errstr(err),
				"type":  "service",
			},
		}

		var res response
		obj := reflect.ValueOf(rsp)
		elem := obj.Elem()
		if nil == err && nil != rsp && elem.Kind() == reflect.Struct {
			rescode := elem.FieldByName("Rescode")
			if rescode.Kind() == reflect.Interface {
				res.rescode = rescode.Interface()
			}
			resmsg := elem.FieldByName("Resmsg")
			if resmsg.Kind() == reflect.String || resmsg.Kind() == reflect.Interface {
				res.resmsg = resmsg.Interface()
			}

			extra := elem.FieldByName("Extra")
			if extra.Kind() == reflect.Interface {
				res.extra = extra.Interface()
			}
		}

		// 上报数据到统计服务
		var fres foot.RPCFootRes
		ferr := HttpRequest(config.Get("statis", "svrname").String("footnode"), config.Get("static", "rpcmethod").String("/foot/rpc"), freq, &fres, "application/proto")
		logger.Infof("[Log Wrapper]-%v, Req: %v, Res: %s, from: %v, ip: %v, errinfo: %v, ferrinfo: %v\n", req.Method(), req.Body(), res, md["Remote"], md["Local"], err, ferr)
		return err
	}
}

// 获取服务对象
//
func NewService() micro.Service {
	hystrixsrc.DefaultTimeout = config.Get("timeout").Int(3) * 1000

	// Create a new service. Optionally include some options here.
	// 设置限流，设置能同时处理的请求数，超过这个数就不继续处理
	br := ratelimit.NewBucketWithRate(float64(config.Get("rate").Int(1000)), int64(config.Get("rate").Int(1000)+200))

	regis := s2s.NewRegistry(
		registry.Addrs(config.Get("s2s", "address").String("")),
		registry.Secure(config.Get("s2s", "secure").Bool(false)),
	)
	svr := micro.NewService(
		micro.Name(config.Get("name").String("")),
		micro.Transport(grpc.NewTransport()),
		micro.Registry(regis),
		micro.Version(config.Get("version").String("0.0.1")),

		// 设置熔断,超过默认值就直接不发送请求
		// 可以通过 github.com/afex/hystrix-go/hystrix设置默认值
		// 超时时间和并发数
		// 所有从此节点发出的Micro服务调用都会受到熔断插件的限制和保护。
		// 熔断是调用级别的
		// doc:https://medium.com/@dche423/micro-in-action-7-cn-ce75d5847ef4
		// 熔断功能作用于客户端，设置恰当阈值以后， 它可以保障客户端资源不会被耗尽
		// —— 哪怕是它所依赖的服务处于不健康的状态，也会快速返回错误，而不是让调用方长时间等待。
		micro.WrapClient(hystrix.NewClientWrapper()),
		// 用于限流限频
		// 与熔断类似， 限流也是分布式系统中常用的功能。
		// 不同的是， 限流在服务端生效，它的作用是保护服务器： 在请求处理速度达到设定的限制以后，
		// 便不再接收和处理更多新请求，直到原有请求处理完成， 腾出空闲。 避免服务器因为客户端的疯狂调用而整体垮掉。
		micro.WrapClient(ratelimiter.NewClientWrapper(br, false)),
		micro.WrapHandler(ratelimiter.NewHandlerWrapper(br, false)),

		// 客户端调用跟踪，每个请求调用之前都会调用这个中间件函数
		micro.WrapCall(metricsWrap),
		// 服务端被调跟踪，每个请求被处理之前都会调用这个中间件函数
		micro.WrapHandler(logWrapper),
		micro.BeforeStop(func() error {
			if nil == s2s.GetDeregister().LocalSvr {
				return nil
			}

			regis.Deregister(s2s.GetDeregister().LocalSvr)

			// 等待3秒结束
			logger.Info("Waitting 3 seconed over!")
			for i := 0; i < 3; i++ {
				logger.Info("Waitting 3 seconed over! ........ ", 3-i)
				time.Sleep(1 * time.Second)
			}

			return nil
		}),
	)

	svr.Init()

	return svr
}

// 获取http服务对象
//
// @return micro.Service
//
func HttpService(router *gin.Engine) micro.Service {
	hystrixsrc.DefaultTimeout = config.Get("timeout").Int(3) * 1000

	// Create a new service. Optionally include some options here.
	// 设置限流，设置能同时处理的请求数，超过这个数就不继续处理
	br := ratelimit.NewBucketWithRate(float64(config.Get("rate").Int(1000)), int64(config.Get("rate").Int(1000)+200))

	srv := httpServer.NewServer(
		server.Name(config.Get("name").String("")),
		server.Version(config.Get("version").String("0.0.1")),
	)

	hd := srv.NewHandler(router)
	err := srv.Handle(hd)
	if nil != err {
		panic(err)
	}

	regis := s2s.NewRegistry(
		registry.Addrs(config.Get("s2s", "address").String("")),
		registry.Secure(config.Get("s2s", "secure").Bool(false)),
	)
	svrice := micro.NewService(
		micro.Server(srv),
		micro.Registry(regis),
		// 客户端调用跟踪，每个请求调用之前都会调用这个中间件函数
		micro.WrapCall(metricsWrap),
		// 服务端被调跟踪，每个请求被处理之前都会调用这个中间件函数
		micro.WrapHandler(logWrapper),
		// 设置熔断,超过默认值就直接不发送请求
		// 可以通过 github.com/afex/hystrix-go/hystrix设置默认值
		// 超时时间和并发数
		// 所有从此节点发出的Micro服务调用都会受到熔断插件的限制和保护。
		// 熔断是调用级别的
		// doc:https://medium.com/@dche423/micro-in-action-7-cn-ce75d5847ef4
		// 熔断功能作用于客户端，设置恰当阈值以后， 它可以保障客户端资源不会被耗尽
		// —— 哪怕是它所依赖的服务处于不健康的状态，也会快速返回错误，而不是让调用方长时间等待。
		micro.WrapClient(hystrix.NewClientWrapper()),
		// 用于限流限频
		// 与熔断类似， 限流也是分布式系统中常用的功能。
		// 不同的是， 限流在服务端生效，它的作用是保护服务器： 在请求处理速度达到设定的限制以后，
		// 便不再接收和处理更多新请求，直到原有请求处理完成， 腾出空闲。 避免服务器因为客户端的疯狂调用而整体垮掉。
		micro.WrapClient(ratelimiter.NewClientWrapper(br, false)),
		micro.WrapHandler(ratelimiter.NewHandlerWrapper(br, false)),
		micro.BeforeStop(func() error {
			if nil == s2s.GetDeregister().LocalSvr {
				return nil
			}

			// 等待3秒结束
			regis.Deregister(s2s.GetDeregister().LocalSvr)
			logger.Info("Waitting 3 seconed over!")
			for i := 0; i < 3; i++ {
				logger.Info("Waitting 3 seconed over! ........ ", 3-i)
				time.Sleep(1 * time.Second)
			}

			return nil
		}),
	)

	svrice.Init()
	return svrice
}

// 获取http服务中对数据的编码器
// 主要用来编解码HTTP服务数据
//
// @param contentType 	数据类型
// @return {codec, err}
//
func HttpCodec(contentType string) (codec Codec, err error) {
	if 0 == len(contentType) {
		err = errors.New("contentType is nil")

		return
	}

	if _, ok := defaultHTTPCodecs[contentType]; !ok {
		err = errors.New("Not support codec. only support application/json,proto,protobuf and octet-stream.")

		return
	}

	codec = defaultHTTPCodecs[contentType]
	return
}

// 获取http客户端对象
//
// @return Client
//
func HttpClient() client.Client {
	regis := s2s.NewRegistry(
		registry.Addrs(config.Get("s2s", "address").String("")),
		registry.Secure(config.Get("s2s", "secure").Bool(false)),
	)

	s := selector.NewSelector(selector.Registry(regis))
	httpcli := httpClient.NewClient(client.Selector(s))
	return httpcli
}

// 发起http请求
//
// @param 	svrname		服务名
// @param 	method 		调用方法名或路径名
// @param 	request 	请求体
// @param 	response 	响应数据
// @param 	contentType	请求数据类型
// @return 	{error}
//
func HttpRequest(svrname, method string, request, response interface{}, contentType string, address ...string) (err error) {
	defer func() {
		logger.Info("HttpRequest", "svrname: "+svrname, "method: "+method, "contentType: ", contentType)
	}()

	if 0 == len(svrname) || 0 == len(method) {
		logger.Error("HttpRequest param errror")

		return
	}

	if 0 == len(contentType) {
		err = errors.New("contentType is nil")

		return
	}

	if _, ok := defaultHTTPCodecs[contentType]; !ok {
		err = errors.New("Not support codec. only support application/json,proto,protobuf and octet-stream.")

		return
	}

	cli := HttpClient()
	req := cli.NewRequest(svrname, method, request, client.WithContentType(contentType))
	if 0 < len(address) {
		err = cli.Call(context.Background(), req, response, client.WithAddress(address...))
		if nil != err {
			return
		}
	} else {
		err = cli.Call(context.Background(), req, response)
		if nil != err {
			return
		}
	}

	return
}
