package main

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/connection"
	"github.com/obgnail/go-frp/consts"
	"github.com/obgnail/go-frp/utils"
	"io"
	"log"
	"time"
)

var heartbeatTimer *time.Timer = nil

type ProxyClient struct {
	ProxyName  string
	LocalPort  int64
	RemoteAddr string
	RemotePort int64

	connChan chan *connection.Conn
}

func NewProxyClient(name string, localPort int64, remoteAddr string, remotePort int64) (*ProxyClient, error) {
	tcpConn, err := utils.ConnectServer(remoteAddr, remotePort)
	if err != nil {
		return nil, errors.Trace(err)
	}
	c := connection.NewConn(tcpConn)
	pc := &ProxyClient{
		ProxyName:  name,
		LocalPort:  localPort,
		RemoteAddr: remoteAddr,
		RemotePort: remotePort,
		connChan:   make(chan *connection.Conn, 1),
	}
	pc.connChan <- c
	return pc, nil
}

func (c *ProxyClient) SendHeartbeatMsg(conn *connection.Conn, msg *consts.Message) {
	fmt.Printf("receive msg:%+v\n", msg)

	heartbeatTimer.Reset(consts.HeartbeatTimeout)
	time.Sleep(10 * time.Second)
	resp := consts.NewMessage(consts.TypeClientWaitHeartbeat, "", c.ProxyName, nil)
	err := conn.SendMessage(resp)
	if err != nil {
		log.Println("[WARN] server write heartbeat err", errors.Trace(err))
	}
}

func (c *ProxyClient) GetLocalConn() (localConn *connection.Conn, err error) {
	tcpConn, err := utils.ConnectServer("127.0.0.1", c.LocalPort)
	if err != nil {
		log.Println("[Error] get local conn err", errors.Trace(err))
		return
	}
	localConn = connection.NewConn(tcpConn)
	return
}

func (c *ProxyClient) GetRemoteConn(remoteAddr string, remotePort int64) (remoteConn *connection.Conn, err error) {
	tcpConn, err := utils.ConnectServer(remoteAddr, remotePort)
	if err != nil {
		log.Println("[Error] get remote conn err", errors.Trace(err))
		return
	}
	remoteConn = connection.NewConn(tcpConn)
	return
}

func (c *ProxyClient) JoinConn(serverConn *connection.Conn, msg *consts.Message) {
	localConn, err := c.GetLocalConn()
	if err != nil {
		log.Printf("[ERROR] ProxyName [%s], get local conn error, %v", c.ProxyName, errors.Trace(err))
		return
	}
	remoteConn, err := c.GetRemoteConn(c.RemoteAddr, c.RemotePort)
	if err != nil {
		log.Printf("[ERROR] ProxyName [%s], get remote conn error, %v", c.ProxyName, errors.Trace(err))
		return
	}

	joinMsg := consts.NewMessage(consts.TypeProxyClientWaitProxyServer, "", c.ProxyName, nil)
	err = remoteConn.SendMessage(joinMsg)
	if err != nil {
		log.Printf("[ERROR] ProxyName [%s], write to server error, %v", c.ProxyName, err)
		return
	}
	log.Printf("Join two conns, (l[%s] r[%s]) (l[%s] r[%s])\n", localConn.GetLocalAddr(), localConn.GetRemoteAddr(),
		remoteConn.GetLocalAddr(), remoteConn.GetRemoteAddr())
	go connection.Join(localConn, remoteConn)
}

func (c *ProxyClient) Run() {
	serverConn, ok := <-c.connChan
	if !ok {
		log.Fatal("[Error] has no conn")
	}
	msg := consts.NewMessage(consts.TypeClientInit, "", c.ProxyName, nil)
	if err := serverConn.SendMessage(msg); err != nil {
		log.Println("[WARN] client write init msg err", errors.Trace(err))
		return
	}
	heartbeatTimer = time.AfterFunc(consts.HeartbeatTimeout, func() {
		log.Println("[WARN] Heartbeat timeout!")
		if serverConn != nil {
			serverConn.Close()
		}
	})
	defer heartbeatTimer.Stop()

	for {
		msg, err := serverConn.ReadMessage()
		if err != nil {
			log.Println("[WARN] proxy client read err:", errors.Trace(err))
			if err == io.EOF {
				log.Printf("ProxyName [%s], server is dead!\n", c.ProxyName)
				return
			}
			continue
		}

		switch msg.Type {
		case consts.TypeServerWaitHeartbeat:
			go c.SendHeartbeatMsg(serverConn, msg)
		case consts.TypeProxyServerWaitProxyClient:
			go c.JoinConn(serverConn, msg)
		}
	}
}
