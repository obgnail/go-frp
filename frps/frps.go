package main

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/connection"
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

// 所有连接发送的数据都会到handler函数处理
func (s *ProxyServer) Process(conn *connection.Conn) {
	for {
		req, err := conn.ReadRequest()
		if err != nil {
			log.Println("[WARN] proxy server read request err:", errors.Trace(err))
			if err == io.EOF {
				log.Printf("ProxyName [%s], client is dead!\n", s.Name)
				conn.Close()
			}
			return
		}
		log.Println("sever receive:", req.ProxyName, req.Type)
		time.Sleep(time.Second)
		resp := connection.NewHeartbeatResponse(req.ProxyName)
		conn.WriteResponse(resp)
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
