package main

import (
	"fmt"
	"github.com/obgnail/go-frp/consts"
)

func main() {
	fmt.Println("--- client start ---")

	var appClientList = []*consts.AppClient{
		{Name: "SSH", LocalPort: 22},
	}

	proxyClient, err := NewProxyClient("common", 5555, "0.0.0.0", 8888, appClientList)
	if err != nil {
		fmt.Println(err)
	}
	proxyClient.Run()
}
