package main

import (
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/connection"
	"github.com/obgnail/go-frp/consts"
	"github.com/obgnail/go-frp/e"
	log "github.com/sirupsen/logrus"
	"io"
	"sync"
	"time"
)

type ServerStatus int

const (
	Idle ServerStatus = iota
	Ready
	Work
)

// commonServer: Used to establish connection and keep heartbeat。
// appServer(appProxyServer): Used to proxy data
type ProxyServer struct {
	Name         string
	bindAddr     string
	listenPort   int64
	appServerMap map[string]*consts.AppServer

	listener              *connection.Listener
	status                ServerStatus            // used in appServer only
	waitToJoinUserConnMap sync.Map                // map[appServerName]UserConn, used in appServer only
	onListenAppServers    map[string]*ProxyServer // appServer which is listening its own port, used in commonServer only
	heartbeatChan         chan *consts.Message    // when get heartbeat msg, put msg in, used in commonServer only
}

func NewProxyServer(name, bindAddr string, listenPort int64, appProxyList []*consts.AppServer) (*ProxyServer, error) {
	tcpListener, err := connection.NewListener(bindAddr, listenPort)
	if err != nil {
		return nil, errors.Trace(err)
	}
	appServerMap := make(map[string]*consts.AppServer, len(appProxyList))
	for _, app := range appProxyList {
		appServerMap[app.Name] = app
	}
	ps := &ProxyServer{
		Name:               name,
		bindAddr:           bindAddr,
		listenPort:         listenPort,
		appServerMap:       appServerMap,
		status:             Idle,
		listener:           tcpListener,
		onListenAppServers: make(map[string]*ProxyServer, len(appProxyList)),
		heartbeatChan:      make(chan *consts.Message, 1),
	}
	return ps, nil
}

func (s *ProxyServer) SetStatus(status ServerStatus) {
	s.status = status
}

func (s *ProxyServer) GetStatus() ServerStatus {
	return s.status
}

func (s *ProxyServer) CloseClient(clientConn *connection.Conn) {
	log.Info("close conn: ", clientConn.String())
	clientConn.Close()
	for _, app := range s.onListenAppServers {
		app.listener.Close()
	}

	// clear all
	s.onListenAppServers = make(map[string]*ProxyServer, len(s.onListenAppServers))
}

func (s *ProxyServer) checkApp(msg *consts.Message) (map[string]*consts.AppServer, error) {
	if msg.Meta == nil {
		return nil, e.EmptyError(e.ModelMessage, e.Meta)
	}
	wantProxyApps := make(map[string]*consts.AppClient)
	for name, app := range msg.Meta.(map[string]interface{}) {
		a := app.(map[string]interface{})
		wantProxyApps[name] = &consts.AppClient{
			Name:      a["Name"].(string),
			LocalPort: int64(a["LocalPort"].(float64)),
			Password:  a["Password"].(string),
		}
	}
	waitToListenAppServers := make(map[string]*consts.AppServer)
	for _, appClient := range wantProxyApps {
		appServer, ok := s.appServerMap[appClient.Name]
		if !ok {
			return nil, e.NotFoundError(e.ModelServer, e.App)
		}
		if appClient.Password != appServer.Password {
			return nil, e.InvalidPasswordError(appClient.Name)
		}
		port, err := connection.TryGetFreePort(5)
		if err != nil {
			return nil, errors.Trace(err)
		}
		appServer.ListenPort = int64(port)
		waitToListenAppServers[appClient.Name] = appServer
	}
	return waitToListenAppServers, nil
}

func (s *ProxyServer) initApp(clientConn *connection.Conn, msg *consts.Message) {
	waitToListenAppServers, err := s.checkApp(msg)
	if err != nil {
		err = errors.Trace(err)
		log.Error(errors.ErrorStack(err))
		s.CloseClient(clientConn)
		return
	}

	// 开始代理具体服务
	for _, app := range waitToListenAppServers {
		go s.startProxyApp(clientConn, app)
	}

	// 告知client这些App可以进行代理
	resp := consts.NewMessage(consts.TypeAppMsg, "", s.Name, waitToListenAppServers)
	err = clientConn.SendMessage(resp)
	if err != nil {
		log.Error(errors.ErrorStack(errors.Trace(err)))
		s.CloseClient(clientConn)
		return
	}

	// keep Heartbeat
	go func() {
		for {
			select {
			case <-s.heartbeatChan:
				log.Debug("received heartbeat msg from", clientConn.GetRemoteAddr())
				resp := consts.NewMessage(consts.TypeServerHeartbeat, "", s.Name, nil)
				err := clientConn.SendMessage(resp)
				if err != nil {
					log.Warn(e.SendHeartbeatMessageError())
					log.Warn(errors.ErrorStack(errors.Trace(err)))
					return
				}
			case <-time.After(consts.HeartbeatTimeout):
				log.Errorf("ProxyName [%s], user conn [%s] Heartbeat timeout", s.Name, clientConn.GetRemoteAddr())
				if clientConn != nil {
					s.CloseClient(clientConn)
				}
				return
			}
		}
	}()
}

func (s *ProxyServer) startProxyApp(clientConn *connection.Conn, app *consts.AppServer) {
	if ps, ok := s.onListenAppServers[app.Name]; ok {
		ps.listener.Close()
	}

	appProxyServer, err := NewProxyServer(app.Name, s.bindAddr, app.ListenPort, nil)
	if err != nil {
		log.Error(errors.ErrorStack(errors.Trace(err)))
		return
	}
	s.onListenAppServers[app.Name] = appProxyServer

	for {
		conn, err := appProxyServer.listener.GetConn()
		if err != nil {
			log.Error(errors.ErrorStack(errors.Trace(err)))
			return
		}
		log.Info("user connect success:", conn.String())

		// connection from client
		if appProxyServer.GetStatus() == Ready && conn.GetRemoteIP() == clientConn.GetRemoteIP() {
			msg, err := conn.ReadMessage()
			if err != nil {
				log.Warnf("proxy client read err:", errors.Trace(err))
				if err == io.EOF {
					log.Errorf("ProxyName [%s], server is dead!", appProxyServer.Name)
					s.CloseClient(conn)
					return
				}
				continue
			}
			if msg.Type != consts.TypeClientJoin {
				log.Warn("get wrong msg")
				continue
			}

			appProxyPort := msg.Content
			newClientConn, ok := s.waitToJoinUserConnMap.Load(appProxyPort)
			if !ok {
				log.Error("waitToJoinUserConnMap load failed. appProxyAddrEny:", appProxyPort)
				continue
			}
			s.waitToJoinUserConnMap.Delete(appProxyPort)

			waitToJoinClientConn := conn
			waitToJoinUserConn := newClientConn.(*connection.Conn)
			log.Infof("Join two connections, [%s] <====> [%s]", waitToJoinUserConn.String(), waitToJoinClientConn.String())
			go connection.Join(waitToJoinUserConn, waitToJoinClientConn)
			appProxyServer.SetStatus(Work)

			// connection from user
		} else {
			s.waitToJoinUserConnMap.Store(app.Name, conn)
			time.AfterFunc(consts.JoinConnTimeout, func() {
				uc, ok := s.waitToJoinUserConnMap.Load(app.Name)
				if !ok || uc == nil {
					return
				}
				if conn == uc.(*connection.Conn) {
					log.Errorf("ProxyName [%s], user conn [%s], join connections timeout", s.Name, conn.GetRemoteAddr())
					conn.Close()
				}
				appProxyServer.SetStatus(Idle)
			})

			// 通知client, Dial到此端口
			msg := consts.NewMessage(consts.TypeAppWaitJoin, app.Name, app.Name, nil)
			err := clientConn.SendMessage(msg)
			if err != nil {
				log.Warn(errors.ErrorStack(errors.Trace(err)))
				return
			}
			appProxyServer.SetStatus(Ready)
		}
	}
}

// 所有连接发送的控制数据都会到此函数处理
func (s *ProxyServer) process(clientConn *connection.Conn) {
	for {
		msg, err := clientConn.ReadMessage()
		if err != nil {
			log.Warn(errors.ErrorStack(errors.Trace(err)))
			if err == io.EOF {
				log.Infof("ProxyName [%s], client is dead!", s.Name)
				s.CloseClient(clientConn)
			}
			return
		}

		switch msg.Type {
		case consts.TypeInitApp:
			go s.initApp(clientConn, msg)
		case consts.TypeClientHeartbeat:
			s.heartbeatChan <- msg
		}
	}
}

func (s *ProxyServer) Run() {
	if s == nil {
		err := e.EmptyError(e.ModelServer, e.Server)
		log.Fatal(err)
	}
	if s.listener == nil {
		err := e.EmptyError(e.ModelServer, e.Listener)
		log.Fatal(err)
	}
	for {
		clientConn, err := s.listener.GetConn()
		if err != nil {
			log.Warn("proxy get conn err:", errors.Trace(err))
			continue
		}
		go s.process(clientConn)
	}
}
