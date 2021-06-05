package service

import (
	"context"
	"time"

	"github.com/asim/go-micro/plugins/wrapper/breaker/hystrix/v3"
	ratelimiter "github.com/asim/go-micro/plugins/wrapper/ratelimiter/ratelimit/v3"

	grpc "github.com/asim/go-micro/plugins/transport/grpc/v3"

	s2s "github.com/heegspace/heegrpc/registry"

	registry "github.com/asim/go-micro/v3/registry"

	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/server"
	"github.com/juju/ratelimit"

	"github.com/micro/go-micro/v2/config"
)

// 客户端调用追踪
func metricsWrap(cf client.CallFunc) client.CallFunc {
	return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
		t := time.Now()
		err := cf(ctx, node, req, rsp, opts)

		logger.Infof("[Metrics Wrapper]Node: %v, Service: %v,  Endpoint: %s, err: %v, duration: %v\n", node, req.Service(), req.Endpoint(), err, time.Since(t))
		return err
	}
}

// 服务端日志跟踪
func logWrapper(fn server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp interface{}) error {
		err := fn(ctx, req, rsp)
		logger.Infof("[Log Wrapper] Endpoint: %s,  method: %s, errinfo: %v", req.Endpoint(), req.Method(), err)

		return err
	}
}

func NewService() micro.Service {
	// Create a new service. Optionally include some options here.
	// 设置限流，设置能同时处理的请求数，超过这个数就不继续处理
	br := ratelimit.NewBucketWithRate(float64(config.Get("rate").Int(1000)), int64(config.Get("rate").Int(1000)+200))

	regis := s2s.NewRegistry(
		registry.Addrs(config.Get("s2s", "address").String("")),
		registry.Secure(config.Get("s2s", "secure").String("http")),
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
	)

	svr.Init()

	return svr
}
