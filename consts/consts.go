package consts

import "time"

const (
	HeartbeatTimeout = 30 * time.Second
	UserConnTimeout  = 30 * time.Second
)

type AppServer struct {
	Name       string
	ListenPort int64
}

type AppClient struct {
	Name      string
	LocalPort int64
}
