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
	Work
)

var heartbeatTimer *time.Timer = nil

type ProxyServer struct {
	Name       string
	BindAddr   string
	ListenPort int64
	Status     ServerStatus

	listener       *connection.Listener
	clientConnChan chan *connection.Conn
	mutex          sync.Mutex
}

func NewProxyServer(name, bindAddr string, listenPort int64) (*ProxyServer, error) {
	tcpListener, err := connection.NewListener(bindAddr, listenPort)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ps := &ProxyServer{
		Name:           name,
		BindAddr:       bindAddr,
		ListenPort:     listenPort,
		Status:         Idle,
		listener:       tcpListener,
		clientConnChan: make(chan *connection.Conn),
	}
	return ps, nil
}

func (s *ProxyServer) Lock() {
	s.mutex.Lock()
}

func (s *ProxyServer) Unlock() {
	s.mutex.Unlock()
}

func (s *ProxyServer) SendHeartbeatMsg(conn *connection.Conn, msg *consts.Message) {
	fmt.Printf("receive msg:%+v\n", msg)
	heartbeatTimer.Reset(consts.HeartbeatTimeout)

	resp := consts.NewMessage(consts.TypeServerWaitHeartbeat, "", s.Name, nil)
	err := conn.SendMessage(resp)
	if err != nil {
		log.Println("[WARN] server write heartbeat response err", errors.Trace(err))
	}
}

func (s *ProxyServer) HandlerClientInit(conn *connection.Conn, msg *consts.Message) {
	heartbeatTimer = time.AfterFunc(consts.HeartbeatTimeout, func() {
		log.Println("[WARN] Heartbeat timeout!")
		if conn != nil {
			conn.Close()
		}
	})
	defer heartbeatTimer.Stop()

	fmt.Printf("%+v\n", msg)

	resp := consts.NewMessage(consts.TypeServerWaitHeartbeat, "proxy started", s.Name, nil)
	err := conn.SendMessage(resp)
	if err != nil {
		log.Println("[WARN] server write heartbeat response err", errors.Trace(err))
	}
}

// 所有连接发送的数据都会到handler函数处理
func (s *ProxyServer) Process(conn *connection.Conn) {
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("[WARN] proxy server read err:", errors.Trace(err))
			if err == io.EOF {
				log.Printf("ProxyName [%s], client is dead!\n", s.Name)
				return
			}
			continue
		}

		switch msg.Type {
		case consts.TypeClientInit:
			go s.HandlerClientInit(conn, msg)
		case consts.TypeClientWaitHeartbeat:
			go s.SendHeartbeatMsg(conn, msg)
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
		go s.Process(conn)
	}
}
