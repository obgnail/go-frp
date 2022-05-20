package main

import (
	"github.com/obgnail/go-frp/consts"
)

func main() {
	NewProxyClient(
		"common",
		5555,
		"127.0.0.1",
		8888,
		[]*consts.AppInfo{
			{Name: "SSH", LocalPort: 22, Password: ""},
			{Name: "HTTP", LocalPort: 7777, Password: ""},
		},
	).Run()
}
