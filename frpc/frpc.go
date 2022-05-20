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
	ProxyName     string
	LocalPort     int64
	RemoteAddr    string
	RemotePort    int64
	wantProxyApps map[string]*consts.AppInfo
	onProxyApps   map[string]*consts.AppInfo
	heartbeatChan chan *consts.Message // when get heartbeat msg, put msg in
}

func NewProxyClient(name string, localPort int64, remoteAddr string, remotePort int64, apps []*consts.AppInfo) *ProxyClient {
	wantProxyApps := make(map[string]*consts.AppInfo)
	for _, app := range apps {
		wantProxyApps[app.Name] = app
	}
	pc := &ProxyClient{
		ProxyName:     name,
		LocalPort:     localPort,
		RemoteAddr:    remoteAddr,
		RemotePort:    remotePort,
		heartbeatChan: make(chan *consts.Message, 1),
		wantProxyApps: wantProxyApps,
		onProxyApps:   make(map[string]*consts.AppInfo, len(apps)),
	}
	return pc
}

func (c *ProxyClient) getJoinConnsFromMsg(msg *consts.Message) (localConn, remoteConn *connection.Conn, err error) {
	appProxyName := msg.Content
	if appProxyName == "" {
		err = e.NotFoundError(e.ModelClient, e.App)
		return
	}
	appServer, ok := c.onProxyApps[appProxyName]
	if !ok {
		err = e.NotFoundError(e.ModelServer, e.App)
		return
	}
	appClient, ok := c.wantProxyApps[appProxyName]
	if !ok {
		err = e.NotFoundError(e.ModelClient, e.Client)
		return
	}

	localConn, err = connection.Dial("127.0.0.1", appClient.LocalPort)
	if err != nil {
		err = errors.Trace(err)
		return
	}

	remoteConn, err = connection.Dial(c.RemoteAddr, appServer.ListenPort)
	if err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *ProxyClient) joinConn(serverConn *connection.Conn, msg *consts.Message) {
	localConn, remoteConn, err := c.getJoinConnsFromMsg(msg)
	if err != nil {
		log.Warnf("get join connections from msg.", errors.ErrorStack(errors.Trace(err)))
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
	if c.wantProxyApps == nil {
		log.Fatal("has no app client to proxy")
	}

	// 通知server开始监听这些app
	msg := consts.NewMessage(consts.TypeInitApp, "", c.ProxyName, c.wantProxyApps)
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
		c.onProxyApps[name] = &consts.AppInfo{
			Name:       appServer["Name"].(string),
			ListenPort: int64(appServer["ListenPort"].(float64)),
		}
	}

	log.Info("---------- Sever ----------")
	for name, app := range c.onProxyApps {
		log.Infof("[%s]:\t%s:%d", name, conn.GetRemoteIP(), app.ListenPort)
	}
	log.Info("---------------------------")

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
				log.Errorf("ProxyName [%s], user conn [%s] Heartbeat timeout", c.ProxyName, conn.GetRemoteAddr())
				if conn != nil {
					conn.Close()
				}
			}
		}
	}()
}

func (c *ProxyClient) Run() {
	tcpConn, err := utils.Dail(c.RemoteAddr, c.RemotePort)
	if err != nil {
		log.Fatal(err)
	}
	conn := connection.NewConn(tcpConn)

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
