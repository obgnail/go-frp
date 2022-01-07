package main

import (
	"fmt"
)

func main() {
	fmt.Println("--- server start ---")
	var appProxyMap = AppProxyMap{
		"SSH": {
			Name:       "SSH",
			BindAddr:   "0.0.0.0",
			ListenPort: 6000,
		},
		"HTTP": {
			Name:       "HTTP",
			BindAddr:   "0.0.0.0",
			ListenPort: 5000,
		},
	}
	commonProxyServer, err := NewProxyServer("common", "0.0.0.0", 8888, appProxyMap)
	if err != nil {
		fmt.Println(err)
	}
	commonProxyServer.Server()
}
