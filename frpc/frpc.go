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

type ProxyClient struct {
	ProxyName    string
	LocalPort    int64
	RemoteAddr   string
	RemotePort   int64
	appClientMap map[string]*consts.AppClient

	onListenAppServers map[string]*consts.AppServer
	connChan           chan *connection.Conn
	heartbeatChan      chan *consts.Message // when get heartbeat msg, put msg in
}

func NewProxyClient(name string, localPort int64, remoteAddr string, remotePort int64, appClientList []*consts.AppClient) (*ProxyClient, error) {
	tcpConn, err := utils.Dail(remoteAddr, remotePort)
	if err != nil {
		return nil, errors.Trace(err)
	}
	appClientMap := make(map[string]*consts.AppClient)
	for _, app := range appClientList {
		appClientMap[app.Name] = app
	}
	pc := &ProxyClient{
		ProxyName:          name,
		LocalPort:          localPort,
		RemoteAddr:         remoteAddr,
		RemotePort:         remotePort,
		connChan:           make(chan *connection.Conn, 1),
		heartbeatChan:      make(chan *consts.Message, 1),
		onListenAppServers: make(map[string]*consts.AppServer),
		appClientMap:       appClientMap,
	}
	pc.connChan <- connection.NewConn(tcpConn)
	return pc, nil
}

func (c *ProxyClient) GetLocalConn(localPort int64) (localConn *connection.Conn, err error) {
	tcpConn, err := utils.Dail("127.0.0.1", localPort)
	if err != nil {
		log.Println("[Error] get local conn err", errors.Trace(err))
		return
	}
	localConn = connection.NewConn(tcpConn)
	return
}

func (c *ProxyClient) GetRemoteConn(remoteAddr string, remotePort int64) (remoteConn *connection.Conn, err error) {
	tcpConn, err := utils.Dail(remoteAddr, remotePort)
	if err != nil {
		log.Println("[Error] get remote conn err", errors.Trace(err))
		return
	}
	remoteConn = connection.NewConn(tcpConn)
	return
}

func (c *ProxyClient) getJoinConnsFromMsg(msg *consts.Message) (localConn, remoteConn *connection.Conn, err error) {
	appProxyName := msg.Content
	if appProxyName == "" {
		err = fmt.Errorf("[ERROR] ProxyName [%s], get app name error")
		return
	}

	appServer, ok := c.onListenAppServers[appProxyName]
	if !ok {
		err = fmt.Errorf("[ERROR] ProxyName [%s], has no such app server, %v", appProxyName, errors.Trace(err))
		return
	}
	appClient, ok := c.appClientMap[appProxyName]
	if !ok {
		err = fmt.Errorf("[ERROR] ProxyName [%s], has no such app client, %v", appProxyName, errors.Trace(err))
		return
	}

	remoteConn, err = c.GetRemoteConn(c.RemoteAddr, appServer.ListenPort)
	if err != nil {
		err = fmt.Errorf("[ERROR] ProxyName [%s], get remote conn error, %v", c.ProxyName, errors.Trace(err))
		return
	}

	localConn, err = c.GetLocalConn(appClient.LocalPort)
	if err != nil {
		err = fmt.Errorf("[ERROR] ProxyName [%s], get local conn error, %v", c.ProxyName, errors.Trace(err))
		return
	}
	return
}

func (c *ProxyClient) JoinConn(serverConn *connection.Conn, msg *consts.Message) {
	localConn, remoteConn, err := c.getJoinConnsFromMsg(msg)
	if err != nil {
		log.Printf("[ERROR] get join conns from msg. %v", errors.Trace(err))
		if localConn != nil {
			localConn.Close()
		}
		if remoteConn != nil {
			localConn.Close()
		}
		return
	}

	joinMsg := consts.NewMessage(consts.TypeClientJoin, msg.Content, c.ProxyName, nil)
	err = remoteConn.SendMessage(joinMsg)
	if err != nil {
		log.Printf("[ERROR] ProxyName [%s], write to server error, %v", c.ProxyName, err)
		return
	}

	log.Printf("Join two conns, (l[%s] -> r[%s]) (l[%s] -> r[%s])\n",
		localConn.GetRemoteAddr(),
		localConn.GetLocalAddr(),
		remoteConn.GetRemoteAddr(),
		remoteConn.GetLocalAddr(),
	)
	go connection.Join(localConn, remoteConn)
}

func (c *ProxyClient) sendInitAppMsg(conn *connection.Conn) {
	if c.appClientMap == nil {
		log.Fatal("[ERROR] has no app client to proxy")
	}

	// 通知server需要c.appClientMap这些app开启监听
	msg := consts.NewMessage(consts.TypeInitApp, "", c.ProxyName, c.appClientMap)
	if err := conn.SendMessage(msg); err != nil {
		log.Println("[WARN] client write init msg err", errors.Trace(err))
		return
	}

	// keep Heartbeat
	go func() {
		for {
			select {
			case <-c.heartbeatChan:
				log.Println("[INFO] received heartbeat msg from", conn.GetRemoteAddr())
				time.Sleep(10 * time.Second)
				resp := consts.NewMessage(consts.TypeClientHeartbeat, "", c.ProxyName, nil)
				err := conn.SendMessage(resp)
				if err != nil {
					log.Println("[WARN] server write heartbeat err", errors.Trace(err))
				}
			case <-time.After(consts.HeartbeatTimeout):
				log.Println("[WARN] Heartbeat timeout!")
				if conn != nil {
					conn.Close()
				}
			}
		}
	}()
}

func (c *ProxyClient) storeServerApp(serverConn *connection.Conn, msg *consts.Message) {
	if msg.Meta == nil {
		log.Fatal("[ERROR] has no app to proxy")
	}

	for name, app := range msg.Meta.(map[string]interface{}) {
		appServer := app.(map[string]interface{})
		c.onListenAppServers[name] = &consts.AppServer{
			Name:       appServer["Name"].(string),
			ListenPort: int64(appServer["ListenPort"].(float64)),
		}
	}

	// prepared, start first heartbeat
	c.heartbeatChan <- msg
}

func (c *ProxyClient) Run() {
	conn, ok := <-c.connChan
	if !ok {
		log.Fatal("[Error] has no conn")
	}

	c.sendInitAppMsg(conn)

	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("[WARN] proxy client read err:", errors.Trace(err))
			if err == io.EOF {
				log.Printf("ProxyName [%s], server is dead!\n", c.ProxyName)
				return
			}
			continue
		}

		switch msg.Type {
		case consts.TypeServerHeartbeat:
			c.heartbeatChan <- msg
		case consts.TypeAppMsg:
			c.storeServerApp(conn, msg)
		case consts.TypeAppWaitJoin:
			go c.JoinConn(conn, msg)
		}
	}
}
