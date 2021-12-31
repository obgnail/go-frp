package frps

import "fmt"

func main() {
	proxyServer, err := NewProxyServer("test", "0.0.0.0", 8888)
	if err != nil {
		fmt.Println(err)
	}
	proxyServer.Server()
}
