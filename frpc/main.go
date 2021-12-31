package frpc

func main() {
	proxyServer, err := NewProxyClient("test", "0.0.0.0", 8888)
	if err != nil {
		fmt.Println(err)
	}
	proxyServer.Server()
}