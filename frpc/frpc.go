package frpc

type ProxyClient struct {
	Name      string
	LocalPort int64
}

func NewProxyClient(name, bindAddr string, listenPort int64) (*ProxyClient, error) {
	return nil, nil
}
