package main

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/connection"
	"github.com/obgnail/go-frp/consts"
	"io"
	"log"
	"sync"
	"time"
)

type ServerStatus int

const (
	Idle ServerStatus = iota
	Ready
	Work
)

// NOTE: ProxySever 在执行 startProxyApp() 时会自我派生,
// 父亲称为 commonServer, 用于建立链接和维持heartbeat。
// 儿子称为 appServer, 用于转发报文。
type ProxyServer struct {
	Name         string
	bindAddr     string
	listenPort   int64
	appServerMap map[string]*consts.AppServer

	listener              *connection.Listener
	onListenAppServers    map[string]*ProxyServer // 正在监听端口的appServer
	status                ServerStatus            // status 字段只用于 appProxy
	waitToJoinUserConnMap sync.Map                // map[appProxyName]UserConn
	heartbeatChan         chan *consts.Message    // when get heartbeat msg, put msg in
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
		onListenAppServers: make(map[string]*ProxyServer),
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

func (s *ProxyServer) checkAppClientFromMsg(msg *consts.Message) map[string]*consts.AppServer {
	if msg.Meta == nil {
		log.Fatal("[ERROR] has no app client to proxy")
	}
	wantProxyApps := make(map[string]*consts.AppClient)
	for name, app := range msg.Meta.(map[string]interface{}) {
		a := app.(map[string]interface{})
		wantProxyApps[name] = &consts.AppClient{
			Name:      a["Name"].(string),
			LocalPort: int64(a["LocalPort"].(float64)),
		}
	}
	waitToListenAppServers := make(map[string]*consts.AppServer)
	for _, appClient := range wantProxyApps {
		appServer, ok := s.appServerMap[appClient.Name]
		if !ok {
			log.Fatal("[ERROR] there is no such app:", appClient.Name)
		}
		waitToListenAppServers[appClient.Name] = appServer
	}
	return waitToListenAppServers
}

func (s *ProxyServer) initApp(clientConn *connection.Conn, msg *consts.Message) {
	waitToListenAppServers := s.checkAppClientFromMsg(msg)

	// 开始代理具体服务
	for _, app := range waitToListenAppServers {
		go s.startProxyApp(clientConn, app)
	}

	// 告知client这些App可以进行代理
	resp := consts.NewMessage(consts.TypeAppMsg, "", s.Name, waitToListenAppServers)
	err := clientConn.SendMessage(resp)
	if err != nil {
		log.Fatal("[WARN] server write app msg response err", errors.Trace(err))
	}

	// keep Heartbeat
	go func() {
		for {
			select {
			case <-s.heartbeatChan:
				log.Println("[INFO] received heartbeat msg from", clientConn.GetRemoteAddr())
				resp := consts.NewMessage(consts.TypeServerHeartbeat, "", s.Name, nil)
				err := clientConn.SendMessage(resp)
				if err != nil {
					log.Println("[WARN] server write heartbeat response err", errors.Trace(err))
				}
			case <-time.After(consts.HeartbeatTimeout):
				log.Println("[WARN] Heartbeat timeout!")
				if clientConn != nil {
					clientConn.Close()
				}
			}
		}
	}()
}

func (s *ProxyServer) startProxyApp(clientConn *connection.Conn, app *consts.AppServer) {
	var appProxyServer *ProxyServer
	if s.onListenAppServers[app.Name] != nil {
		appProxyServer = s.onListenAppServers[app.Name]
	} else {
		ps, err := NewProxyServer(app.Name, s.bindAddr, app.ListenPort, nil)
		if err != nil {
			log.Println("[WARN] start proxy err, maybe address already in use:", errors.Trace(err))
			return
		}
		appProxyServer = ps
	}

	for {
		conn, err := appProxyServer.listener.GetConn()
		if err != nil {
			log.Println("[WARN] proxy get conn err:", errors.Trace(err))
			continue
		}
		log.Printf("[INFO] user connect success: %s -> %s", conn.GetRemoteAddr(), conn.GetLocalAddr())

		// connection from client
		if appProxyServer.GetStatus() == Ready && conn.GetRemoteIP() == clientConn.GetRemoteIP() {
			msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("[WARN] proxy client read err:", errors.Trace(err))
				if err == io.EOF {
					log.Printf("ProxyName [%s], server is dead!\n", appProxyServer.Name)
					return
				}
				continue
			}
			if msg.Type != consts.TypeClientJoin {
				log.Println("[Error] get wrong msg")
				continue
			}

			appProxyPort := msg.Content
			newClientConn, ok := s.waitToJoinUserConnMap.Load(appProxyPort)
			if !ok {
				log.Println("[Error] waitToJoinUserConnMap load failed. appProxyAddrEny:", appProxyPort)
				continue
			}
			s.waitToJoinUserConnMap.Delete(appProxyPort)

			waitToJoinClientConn := conn
			waitToJoinUserConn := newClientConn.(*connection.Conn)
			log.Printf("Join two conns, (l[%s] -> r[%s]) (l[%s] -> r[%s])",
				waitToJoinUserConn.GetRemoteAddr(),
				waitToJoinUserConn.GetLocalAddr(),
				waitToJoinClientConn.GetRemoteAddr(),
				waitToJoinClientConn.GetLocalAddr(),
			)
			go connection.Join(waitToJoinUserConn, waitToJoinClientConn)
			appProxyServer.SetStatus(Work)

			// connection from user
		} else {
			s.waitToJoinUserConnMap.Store(app.Name, conn)
			time.AfterFunc(consts.UserConnTimeout, func() {
				uc, ok := s.waitToJoinUserConnMap.Load(app.Name)
				if !ok || uc == nil {
					return
				}
				if conn == uc.(*connection.Conn) {
					log.Printf("[WARN] ProxyName [%s], user conn [%s] timeout\n", s.Name, conn.GetRemoteAddr())
				} else {
					log.Printf("[INFO] ProxyName [%s], There's another user conn [%s] need to be processed\n", s.Name, conn.GetRemoteAddr())
				}
				appProxyServer.SetStatus(Idle)
			})

			// 通知client, Dial到此端口
			msg := consts.NewMessage(consts.TypeAppWaitJoin, app.Name, app.Name, nil)
			err := clientConn.SendMessage(msg)
			if err != nil {
				log.Println("[WARN] server write response err", errors.Trace(err))
				return
			}
			appProxyServer.SetStatus(Ready)
		}
	}
}

// 所有连接发送的控制数据都会到此函数处理
func (s *ProxyServer) process(conn *connection.Conn) {
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("[WARN] proxy server read err:", errors.Trace(err))
			if err == io.EOF {
				log.Printf("ProxyName [%s], client is dead!\n", s.Name)
				conn.Close()
				return
			}
			log.Println("---- continue ----")
			continue
		}

		switch msg.Type {
		case consts.TypeClientHeartbeat:
			s.heartbeatChan <- msg
		case consts.TypeInitApp:
			go s.initApp(conn, msg)
		}
	}
}

func (s *ProxyServer) Server() {
	if s == nil {
		log.Fatal(fmt.Errorf("proxy server is nil"))
	}
	if s.listener == nil {
		log.Fatal(fmt.Errorf("proxy server has no listener"))
	}
	for {
		conn, err := s.listener.GetConn()
		if err != nil {
			log.Println("[WARN] proxy get conn err:", errors.Trace(err))
			continue
		}
		go s.process(conn)
	}
}
