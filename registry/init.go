package registry

type S2sName struct {
	Host string
	Port int32

	Prority int32 // 当前服务的负载情况，越低负载越低
}
