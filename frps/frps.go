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

type ServerStatus int

const (
	Idle ServerStatus = iota
	Work
)

var heartbeatTimer *time.Timer = nil

var ProxyServerMap = map[string]*ProxyServer{
	"SSH": {
		Name:       "SSH",
		BindAddr:   "0.0.0.0",
		ListenPort: 6000,
		Status:     Idle,
		listener:   nil,
	},
	"HTTP": {
		Name:       "HTTP",
		BindAddr:   "0.0.0.0",
		ListenPort: 5000,
		Status:     Idle,
		listener:   nil,
	},
}

type ProxyServer struct {
	Name       string
	BindAddr   string
	ListenPort int64
	Status     ServerStatus

	listener     *connection.Listener
	userConnList *utils.Queue
}

func NewProxyServer(name, bindAddr string, listenPort int64) (*ProxyServer, error) {
	tcpListener, err := connection.NewListener(bindAddr, listenPort)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ps := &ProxyServer{
		Name:         name,
		BindAddr:     bindAddr,
		ListenPort:   listenPort,
		Status:       Idle,
		listener:     tcpListener,
		userConnList: utils.NewQueue(),
	}
	return ps, nil
}

func (s *ProxyServer) SendHeartbeatMsg(clientConn *connection.Conn, msg *consts.Message) {
	fmt.Printf("receive msg:%+v\n", msg)
	heartbeatTimer.Reset(consts.HeartbeatTimeout)

	resp := consts.NewMessage(consts.TypeServerWaitHeartbeat, "", s.Name, nil)
	err := clientConn.SendMessage(resp)
	if err != nil {
		log.Println("[WARN] server write heartbeat response err", errors.Trace(err))
	}
}

func (s *ProxyServer) HandlerClientInit(clientConn *connection.Conn, msg *consts.Message) {
	ps, ok := ProxyServerMap[msg.ProxyName]
	if !ok {
		log.Fatal("[WARN] has no such proxyName:", msg.ProxyName)
	}

	// start proxy
	proxy, err := NewProxyServer(ps.Name, ps.BindAddr, ps.ListenPort)
	if err != nil {
		log.Println("[ERROR] start proxy err", errors.Trace(err))
		return
	}

	// server User
	go func(clientConn *connection.Conn) {
		for {
			userConn, err := proxy.listener.GetConn()
			log.Printf("[INFO] user connect success: %s -> %s", userConn.GetRemoteAddr(), userConn.GetLocalAddr())
			if err != nil {
				log.Println("[WARN] proxy get conn err:", errors.Trace(err))
				continue
			}
			s.ProcessUserConnection(userConn, clientConn)
		}
	}(clientConn)

	heartbeatTimer = time.AfterFunc(consts.HeartbeatTimeout, func() {
		log.Println("[WARN] Heartbeat timeout!")
		if clientConn != nil {
			clientConn.Close()
		}
	})
	defer heartbeatTimer.Stop()

	resp := consts.NewMessage(consts.TypeServerWaitHeartbeat, "proxy started", s.Name, nil)
	err = clientConn.SendMessage(resp)
	if err != nil {
		log.Println("[WARN] server write heartbeat response err", errors.Trace(err))
	}
}

// 所有连接发送的控制数据和用户数据都会到此函数处理
func (s *ProxyServer) Process(conn *connection.Conn) {
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
			go s.HandlerClientInit(conn, msg)
		case consts.TypeClientWaitHeartbeat:
			go s.SendHeartbeatMsg(conn, msg)
		case consts.TypeProxyClientWaitProxyServer:
			go s.JoinConn(conn, msg)
		}
	}
}

// 所有用户数据都会到此函数处理
func (s *ProxyServer) ProcessUserConnection(userConn *connection.Conn, clientConn *connection.Conn) {
	s.userConnList.Push(userConn)

	msg := consts.NewMessage(consts.TypeProxyServerWaitProxyClient, "", s.Name, nil)
	err := clientConn.SendMessage(msg)
	if err != nil {
		log.Println("[WARN] server write heartbeat response err", errors.Trace(err))
	}
	log.Println("[INFO] Send TypeProxyServerWaitProxyClient success")

	time.AfterFunc(consts.UserConnTimeout, func() {
		uc := s.userConnList.Front()
		if uc == nil {
			return
		}

		if userConn == uc.(*connection.Conn) {
			log.Printf("[WARN] ProxyName [%s], user conn [%s] timeout\n", s.Name, userConn.GetRemoteAddr())
		} else {
			log.Printf("[INFO] ProxyName [%s], There's another user conn [%s] need to be processed\n", s.Name, userConn.GetRemoteAddr())
		}
	})
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
		go s.Process(conn)
	}
}

func (s *ProxyServer) JoinConn(newClientConn *connection.Conn, msg *consts.Message) {
	if s.userConnList.Len() == 0 {
		log.Printf("[ERROR] get user conn from UserConnList: len(s.userConnList) == 0 ")
		newClientConn.Close()
		return
	}
	userConn := s.userConnList.Front().(*connection.Conn)
	if userConn == nil {
		log.Printf("[ERROR] userConn is nil")
		newClientConn.Close()
		return
	}
	s.userConnList.Pop()

	log.Printf("Join two conns, (l[%s] -> r[%s]) (l[%s] -> r[%s])", newClientConn.GetRemoteAddr(), newClientConn.GetLocalAddr(),
		userConn.GetRemoteAddr(), userConn.GetLocalAddr())
	go connection.Join(newClientConn, userConn)
}
