## 模仿frp的基于代理的内网穿透玩具

因为原版frp的可读性很差，故重构。



## Usage

Server：

```bash
cd frps/
go run *.go
```



Client：

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

