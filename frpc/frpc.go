package main

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/connection"
	"github.com/obgnail/go-frp/utils"
	"log"
	"time"
)

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
	conn := connection.NewConn(tcpConn)
	pc := &ProxyClient{
		ProxyName:  name,
		LocalPort:  localPort,
		RemoteAddr: remoteAddr,
		RemotePort: remotePort,
		connChan:   make(chan *connection.Conn, 1),
	}
	pc.connChan <- conn
	return pc, nil
}

func (c *ProxyClient) Run() {
	conn, ok := <-c.connChan
	if !ok {
		log.Fatal("[Error] has no conn")
	}
	req := connection.NewHeartbeatRequest(c.ProxyName)
	log.Printf("222，%+v",req)
	conn.WriteRequest(req)
	log.Println("[INFO] start heartbeat")
	for {
		resp, err := conn.ReadResponse()
		if err != nil {
			log.Println("[WARN] proxy client read response err:", errors.Trace(err))
			//continue
			return
		}
		fmt.Println("client receive:", resp.ProxyName, resp.Type)
		time.Sleep(time.Second)
		req := connection.NewHeartbeatRequest(c.ProxyName)
		conn.WriteRequest(req)
	}

}
