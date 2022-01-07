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

var heartbeatTimer *time.Timer = nil

type AppProxyMap map[string]struct {
	Name       string
	BindAddr   string
	ListenPort int64
}

var appProxyMap = AppProxyMap{
	"SSH": {
		Name:       "SSH",
		BindAddr:   "0.0.0.0",
		ListenPort: 6000,
	},
	"HTTP": {
		Name:       "HTTP",
		BindAddr:   "0.0.0.0",
		ListenPort: 5000,
	},
}

type ProxyServer struct {
	Name       string
	BindAddr   string
	ListenPort int64

	Status ServerStatus

	appProxyMap           AppProxyMap
	listener              *connection.Listener
	waitToJoinUserConnMap sync.Map // map[appProxyPort]UserConn
}

func NewProxyServer(name, bindAddr string, listenPort int64, appProxyMap AppProxyMap) (*ProxyServer, error) {
	tcpListener, err := connection.NewListener(bindAddr, listenPort)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ps := &ProxyServer{
		Name:        name,
		BindAddr:    bindAddr,
		ListenPort:  listenPort,
		appProxyMap: appProxyMap,
		Status:      Idle,
		listener:    tcpListener,
	}
	return ps, nil
}

func (s *ProxyServer) SetStatus(status ServerStatus) {
	s.Status = status
}

func (s *ProxyServer) GetStatus() ServerStatus {
	return s.Status
}

func (s *ProxyServer) keepHeartbeat(clientConn *connection.Conn, msg *consts.Message) {
	fmt.Printf("receive msg:%+v\n", msg)
	if heartbeatTimer == nil {
		log.Fatal("heartbeatTimer is nil")
	}
	heartbeatTimer.Reset(consts.HeartbeatTimeout)

	resp := consts.NewMessage(consts.TypeServerWaitHeartbeat, "", s.Name, nil)
	err := clientConn.SendMessage(resp)
	if err != nil {
		log.Println("[WARN] server write heartbeat response err", errors.Trace(err))
	}
}

func (s *ProxyServer) initClient(clientConn *connection.Conn, msg *consts.Message) {
	// 开始代理具体服务
	go s.startAppProxyServer(clientConn, msg)

	// start first heartbeat
	heartbeatTimer = time.AfterFunc(consts.HeartbeatTimeout, func() {
		log.Println("[WARN] Heartbeat timeout!")
		if clientConn != nil {
			clientConn.Close()
		}
	})
	resp := consts.NewMessage(consts.TypeServerWaitHeartbeat, "proxy started", s.Name, nil)
	err := clientConn.SendMessage(resp)
	if err != nil {
		log.Println("[WARN] server write heartbeat response err", errors.Trace(err))
	}
}

// 所有连接发送的控制数据和用户数据都会到此函数处理
func (s *ProxyServer) process(conn *connection.Conn) {
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("[WARN] proxy server read err:", errors.Trace(err))
			if err == io.EOF {
				log.Printf("ProxyName [%s], client is dead!\n", s.Name)
				return
			}
			log.Println("---- continue ----")
			continue
		}

		switch msg.Type {
		case consts.TypeClientInit:
			go s.initClient(conn, msg)
		case consts.TypeClientWaitHeartbeat:
			go s.keepHeartbeat(conn, msg)
		}
	}
}

func (s *ProxyServer) startAppProxyServer(clientConn *connection.Conn, msg *consts.Message) {
	ps, ok := s.appProxyMap[msg.ProxyName]
	if !ok {
		log.Fatal("[WARN] has no such proxyName:", msg.ProxyName)
	}
	appProxy, err := NewProxyServer(ps.Name, ps.BindAddr, ps.ListenPort, nil)
	if err != nil {
		log.Println("[ERROR] start proxy err", errors.Trace(err))
		return
	}

	for {
		conn, err := appProxy.listener.GetConn()
		if err != nil {
			log.Println("[WARN] proxy get conn err:", errors.Trace(err))
			continue
		}
		log.Printf("[INFO] user connect success: %s -> %s", conn.GetRemoteAddr(), conn.GetLocalAddr())

		// connection from client
		if appProxy.GetStatus() == Ready && conn.GetRemoteIP() == clientConn.GetRemoteIP() {
			msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("[WARN] proxy client read err:", errors.Trace(err))
				if err == io.EOF {
					log.Printf("ProxyName [%s], server is dead!\n", appProxy.Name)
					return
				}
				continue
			}
			if msg.Type != consts.TypeProxyClientWaitProxyServer {
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
			appProxy.SetStatus(Work)

			// connection from user
		} else {
			port := fmt.Sprintf("%d", ps.ListenPort)
			s.waitToJoinUserConnMap.Store(port, conn)

			msg := consts.NewMessage(consts.TypeProxyServerWaitProxyClient, port, s.Name, nil)
			err := clientConn.SendMessage(msg)
			if err != nil {
				log.Println("[WARN] server write heartbeat response err", errors.Trace(err))
			}
			log.Println("[INFO] Send TypeProxyServerWaitProxyClient success")
			appProxy.SetStatus(Ready)

			time.AfterFunc(consts.UserConnTimeout, func() {
				uc, ok := s.waitToJoinUserConnMap.Load(port)
				if !ok || uc == nil {
					return
				}
				if conn == uc.(*connection.Conn) {
					log.Printf("[WARN] ProxyName [%s], user conn [%s] timeout\n", s.Name, conn.GetRemoteAddr())
				} else {
					log.Printf("[INFO] ProxyName [%s], There's another user conn [%s] need to be processed\n", s.Name, conn.GetRemoteAddr())
				}
				appProxy.SetStatus(Idle)
			})
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
