package main

import (
	"fmt"
)

func main() {
	fmt.Println("--- client start ---")
	proxyClient, err := NewProxyClient("test", 9999, "0.0.0.0", 8888)
	if err != nil {
		fmt.Println(err)
	}
	proxyClient.Run()
}
