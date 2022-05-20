package consts

import "time"

const (
	HeartbeatInterval = 10 * time.Second
	HeartbeatTimeout  = 30 * time.Second
	JoinConnTimeout   = 30 * time.Second
)

type AppInfo struct {
	Name       string
	ListenPort int64 // used in server
	LocalPort  int64 // used in local
	Password   string
}
