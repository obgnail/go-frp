package main

import (
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/connection"
	"github.com/obgnail/go-frp/consts"
	"github.com/obgnail/go-frp/e"
	"github.com/obgnail/go-frp/utils"
	log "github.com/sirupsen/logrus"
	"io"
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
		err = errors.Trace(err)
		return
	}
	localConn = connection.NewConn(tcpConn)
	return
}

func (c *ProxyClient) GetRemoteConn(remoteAddr string, remotePort int64) (remoteConn *connection.Conn, err error) {
	tcpConn, err := utils.Dail(remoteAddr, remotePort)
	if err != nil {
		err = errors.Trace(err)
		return
	}
	remoteConn = connection.NewConn(tcpConn)
	return
}

func (c *ProxyClient) getJoinConnsFromMsg(msg *consts.Message) (localConn, remoteConn *connection.Conn, err error) {
	appProxyName := msg.Content
	if appProxyName == "" {
		err = e.NotFoundError(e.ModelClient, e.App)
		return
	}

	appServer, ok := c.onListenAppServers[appProxyName]
	if !ok {
		err = e.NotFoundError(e.ModelServer, e.App)
		return
	}
	appClient, ok := c.appClientMap[appProxyName]
	if !ok {
		err = e.NotFoundError(e.ModelClient, e.Client)
		return
	}

	localConn, err = c.GetLocalConn(appClient.LocalPort)
	if err != nil {
		err = errors.Trace(err)
		return
	}

	remoteConn, err = c.GetRemoteConn(c.RemoteAddr, appServer.ListenPort)
	if err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *ProxyClient) joinConn(serverConn *connection.Conn, msg *consts.Message) {
	localConn, remoteConn, err := c.getJoinConnsFromMsg(msg)
	if err != nil {
		log.Errorf("get join conns from msg.", errors.ErrorStack(errors.Trace(err)))
		if localConn != nil {
			localConn.Close()
		}
		if remoteConn != nil {
			remoteConn.Close()
		}
		return
	}

	joinMsg := consts.NewMessage(consts.TypeClientJoin, msg.Content, c.ProxyName, nil)
	err = remoteConn.SendMessage(joinMsg)
	if err != nil {
		log.Errorf(errors.ErrorStack(errors.Trace(err)))
		return
	}

	log.Infof("Join two connections, [%s] <====> [%s]", localConn.String(), remoteConn.String())
	go connection.Join(localConn, remoteConn)
}

func (c *ProxyClient) sendInitAppMsg(conn *connection.Conn) {
	if c.appClientMap == nil {
		log.Fatal("has no app client to proxy")
	}

	// 通知server开始监听这些app
	msg := consts.NewMessage(consts.TypeInitApp, "", c.ProxyName, c.appClientMap)
	if err := conn.SendMessage(msg); err != nil {
		log.Warn("client write init msg err.", errors.ErrorStack(errors.Trace(err)))
		return
	}
}

func (c *ProxyClient) storeServerApp(conn *connection.Conn, msg *consts.Message) {
	if msg.Meta == nil {
		log.Fatal("has no app to proxy")
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

	// keep Heartbeat
	go func() {
		for {
			select {
			case <-c.heartbeatChan:
				log.Debug("received heartbeat msg from", conn.GetRemoteAddr())
				time.Sleep(consts.HeartbeatInterval)
				resp := consts.NewMessage(consts.TypeClientHeartbeat, "", c.ProxyName, nil)
				err := conn.SendMessage(resp)
				if err != nil {
					log.Warn(e.SendHeartbeatMessageError())
					log.Warn(errors.ErrorStack(errors.Trace(err)))
				}
			case <-time.After(consts.HeartbeatTimeout):
				log.Warnf("ProxyName [%s], user conn [%s] Heartbeat timeout", c.ProxyName, conn.GetRemoteAddr())
				if conn != nil {
					conn.Close()
				}
			}
		}
	}()
}

func (c *ProxyClient) Run() {
	conn, ok := <-c.connChan
	if !ok {
		log.Fatal("has no conn")
	}

	c.sendInitAppMsg(conn)

	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			log.Warn(errors.ErrorStack(errors.Trace(err)))
			if err == io.EOF {
				log.Infof("ProxyName [%s], client is dead!", c.ProxyName)
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
			go c.joinConn(conn, msg)
		}
	}
}
