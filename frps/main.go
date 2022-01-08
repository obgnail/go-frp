package main

import (
	"fmt"
	"github.com/obgnail/go-frp/consts"
)

func main() {
	fmt.Println("--- server start ---")

	appServerList := []*consts.AppServer{
		{Name: "SSH", ListenPort: 6000},
		{Name: "HTTP", ListenPort: 5000},
	}

	commonProxyServer, err := NewProxyServer("common", "0.0.0.0", 8888, appServerList)
	if err != nil {
		fmt.Println(err)
	}
	commonProxyServer.Server()
}
