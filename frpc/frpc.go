package frpc

type IProxyClient interface {
	SendHeartBeatMsg()
}

type ProxyClient struct {
	Name      string
	LocalPort int64
}

