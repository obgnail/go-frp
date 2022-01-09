package consts

import "time"

const (
	HeartbeatInterval = 10 * time.Second
	HeartbeatTimeout  = 30 * time.Second
	JoinConnTimeout   = 30 * time.Second
)

type AppServer struct {
	Name       string
	ListenPort int64
	Password   string
}

type AppClient struct {
	Name      string
	LocalPort int64
	Password  string
}
