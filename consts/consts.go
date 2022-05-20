package consts

import "time"

const (
	HeartbeatInterval = 10 * time.Second
	HeartbeatTimeout  = 30 * time.Second
	JoinConnTimeout   = 30 * time.Second
)

type AppServerInfo struct {
	Name       string
	ListenPort int64
	Password   string
}

type AppInfo struct {
	Name      string
	LocalPort int64
	Password  string
}
