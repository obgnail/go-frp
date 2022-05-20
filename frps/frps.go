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
	Name       string
	bindAddr   string
	listenPort int64
	appInfoMap map[string]*consts.AppServerInfo

	listener              *connection.Listener
	status                ServerStatus            // used in appServer only
	waitToJoinUserConnMap sync.Map                // map[appServerName]UserConn, used in appServer only
	onListenApps          map[string]*ProxyServer // appServer which is listening its own port, used in commonServer only
	heartbeatChan         chan *consts.Message    // when get heartbeat msg, put msg in, used in commonServer only
}

func NewProxyServer(name, bindAddr string, listenPort int64, apps []*consts.AppServerInfo) (*ProxyServer, error) {
	listener, err := connection.NewListener(bindAddr, listenPort)
	if err != nil {
		return nil, errors.Trace(err)
	}
	appInfoMap := make(map[string]*consts.AppServerInfo, len(apps))
	for _, app := range apps {
		appInfoMap[app.Name] = app
	}
	ps := &ProxyServer{
		Name:          name,
		bindAddr:      bindAddr,
		listenPort:    listenPort,
		appInfoMap:    appInfoMap,
		status:        Idle,
		listener:      listener,
		onListenApps:  make(map[string]*ProxyServer, len(apps)),
		heartbeatChan: make(chan *consts.Message, 1),
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
	for _, app := range s.onListenApps {
		app.listener.Close()
	}

	// clear all
	s.onListenApps = make(map[string]*ProxyServer, len(s.onListenApps))
}

func (s *ProxyServer) checkApp(msg *consts.Message) (map[string]*consts.AppServerInfo, error) {
	if msg.Meta == nil {
		return nil, e.EmptyError(e.ModelMessage, e.Meta)
	}
	wantProxyApps := make(map[string]*consts.AppClientInfo)
	for name, app := range msg.Meta.(map[string]interface{}) {
		a := app.(map[string]interface{})
		wantProxyApps[name] = &consts.AppClientInfo{
			Name:      a["Name"].(string),
			LocalPort: int64(a["LocalPort"].(float64)),
			Password:  a["Password"].(string),
		}
	}
	waitToListenAppsInfo := make(map[string]*consts.AppServerInfo)
	for _, appClient := range wantProxyApps {
		appServer, ok := s.appInfoMap[appClient.Name]
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
		waitToListenAppsInfo[appClient.Name] = appServer
	}
	return waitToListenAppsInfo, nil
}

func (s *ProxyServer) initApp(clientConn *connection.Conn, msg *consts.Message) {
	waitToListenAppsInfo, err := s.checkApp(msg)
	if err != nil {
		err = errors.Trace(err)
		log.Error(errors.ErrorStack(err))
		s.CloseClient(clientConn)
		return
	}

	// 开始代理具体服务
	for _, app := range waitToListenAppsInfo {
		go s.startProxyApp(clientConn, app)
	}

	// 告知client这些App可以进行代理
	resp := consts.NewMessage(consts.TypeAppMsg, "", s.Name, waitToListenAppsInfo)
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

func (s *ProxyServer) startProxyApp(clientConn *connection.Conn, app *consts.AppServerInfo) {
	if ps, ok := s.onListenApps[app.Name]; ok {
		ps.listener.Close()
	}

	apps, err := NewProxyServer(app.Name, s.bindAddr, app.ListenPort, nil)
	if err != nil {
		log.Error(errors.ErrorStack(errors.Trace(err)))
		return
	}
	s.onListenApps[app.Name] = apps

	for {
		conn, err := apps.listener.GetConn()
		if err != nil {
			log.Error(errors.ErrorStack(errors.Trace(err)))
			return
		}
		log.Info("user connect success:", conn.String())

		// connection from client
		if apps.GetStatus() == Ready && conn.GetRemoteIP() == clientConn.GetRemoteIP() {
			msg, err := conn.ReadMessage()
			if err != nil {
				log.Warnf("proxy client read err:", errors.Trace(err))
				if err == io.EOF {
					log.Errorf("ProxyName [%s], server is dead!", apps.Name)
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
			apps.SetStatus(Work)

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
				apps.SetStatus(Idle)
			})

			// 通知client, Dial到此端口
			msg := consts.NewMessage(consts.TypeAppWaitJoin, app.Name, app.Name, nil)
			err := clientConn.SendMessage(msg)
			if err != nil {
				log.Warn(errors.ErrorStack(errors.Trace(err)))
				return
			}
			apps.SetStatus(Ready)
		}
	}
}

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

func (s *ProxyServer) Serve() {
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
