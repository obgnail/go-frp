package main

import (
	"fmt"
)


func main() {
	fmt.Println("--- server start ---")
	commonProxyServer, err := NewProxyServer("common", "0.0.0.0", 8888)
	if err != nil {
		fmt.Println(err)
	}
	commonProxyServer.Server()
}
