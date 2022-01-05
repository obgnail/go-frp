package main

import (
	"fmt"
)

func main() {
	fmt.Println("--- client start ---")
	proxyClient, err := NewProxyClient("SSH", 22, "0.0.0.0", 8888)
	if err != nil {
		fmt.Println(err)
	}
	proxyClient.Run()
}
