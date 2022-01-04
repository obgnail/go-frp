package main

import (
	"fmt"
)


func main() {
	fmt.Println("--- server start ---")
	proxyServer, err := NewProxyServer("test", "0.0.0.0", 8888)
	if err != nil {
		fmt.Println(err)
	}
	proxyServer.Server()
}
