## 模仿frp的基于代理的内网穿透玩具

因为原版frp的可读性很差，故重构。



## Usage

Server：

```go
func main() {
	appServerList := []*consts.AppServer{
    {Name: "SSH", ListenPort: 6000, Password: "my_ssh_server_password"},
		{Name: "HTTP", ListenPort: 5000, Password: ""},
	}

	commonProxyServer, err := NewProxyServer("common", "0.0.0.0", 8888, appServerList)
	if err != nil {
		log.Error(errors.ErrorStack(err))
	}
	commonProxyServer.Run()
}
```

```bash
cd frps/
go run *.go
```



Client：

```go
func main() {
	appClientList := []*consts.AppClient{
		{Name: "SSH", LocalPort: 22, Password: "my_ssh_server_password"},
		{Name: "HTTP", LocalPort: 7777, Password: ""},
	}

	commonProxyClient, err := NewProxyClient("common", 5555, "0.0.0.0", 8888, appClientList)
	if err != nil {
		log.Error(errors.ErrorStack(err))
	}
	commonProxyClient.Run()
}
```

```bash
cd frpc/
go run *.go
```



User：

```bash
# ssh: 
# proxy 6000 to 22
ssh -o 6000 yourComputerName@127.0.0.1

# http: 
# proxy 5000 to 7777
python3 -m http.server 7777
curl 127.0.0.1:5000
```

## sequenceDiagram

```mermaid
sequenceDiagram
    participant User
    participant CommonServer
    participant ProxyServer
    participant CommonClient
    participant ProxyClient

    Note over CommonServer, CommonClient: 准备阶段
    CommonServer ->> CommonServer: 监听BindPort
    CommonClient ->> CommonClient: 构建ProxyClient对象
    
    Note over CommonServer, CommonClient: 初始化阶段
    CommonClient ->> + CommonServer: 连接到Common conn，发送所有ProxyClient的信息
    CommonServer ->> CommonServer: 根据ProxyName取出所有的ProxyServer对象，校验
    CommonServer ->> ProxyServer: 通知每个ProxyServer开始监听各自的端口
    ProxyServer ->> ProxyServer: 开启goroutine，监听ProxyConn，等待User连接
    CommonServer ->> CommonServer: 开启goroutine，监听heartbeat
    CommonServer -->> - CommonClient: 告知CommonClient所有ProxyServer的信息，表明Ready，可以进行代理
    CommonClient ->> CommonClient: 存储ProxySever信息
   	par par and loop
   		CommonClient ->> CommonServer: 发送heartbeat
   		CommonServer -->> CommonClient: 响应heartbeat
   	end
    
    Note over User, ProxyClient: 代理阶段
    User ->> + ProxyServer: User连接ProxyServer，希望开始代理
    ProxyServer ->> ProxyServer: 储存UserConn
    ProxyServer ->> CommonServer: 通知CommonServer<br/>将「希望开启代理」的信号传给CommonClient
    CommonServer ->> CommonClient: 告知CommonClient「希望开启代理」
    CommonClient ->> CommonClient: 从存储中找到「希望开启代理」的APP信息
    CommonClient ->> ProxyClient: 开启本地端口
    ProxyClient -->> CommonClient: 返回localConn
    CommonClient ->> ProxyServer: 连接到ProxyServer
    ProxyServer ->> ProxyServer: 1.校验ProxyClientConn和message<br/>2.从存储中找到UserConn
    par
   		ProxyServer ->> ProxyServer: Join ProxyClientConn and UserConn
   	end
		ProxyServer -->> CommonClient: 返回remoteConn
		par
   		CommonClient ->> CommonClient: Join localConn and remoteConn
   	end
    ProxyServer -->> - User: 返回
```



## 两个Join操作

```mermaid
sequenceDiagram
    participant User(Mac)
    participant ProxyServer(Tencent)
    participant ProxyClient(Rasp)
    participant UnixSocket(Rasp)

		Note over User(Mac), ProxyServer(Tencent): UserConn
		Note over ProxyServer(Tencent), ProxyClient(Rasp): ProxyClientConn
		
		Note over ProxyServer(Tencent), ProxyClient(Rasp): remoteConn
		Note over ProxyClient(Rasp), UnixSocket(Rasp): localConn
```

对于Tencent来说：

- UserConn：表示来自于Mac的TCP连接。
- ProxyClientConn：表示连接到Rasp某个服务的TCP连接。

对于Rasp来说：

- RemoteConn：表示来自于Tencent的TCP连接。
- LocalConn：表示本机某个服务的UnixSocket连接。

经过两个Join操作，ProxyClientConn、UserConn、remoteConn、UnixSocketConn都互相串通。也就实现了远程User与UnixSocket的连接，也就是Mac通过Tencent，连接到了Rasp。


