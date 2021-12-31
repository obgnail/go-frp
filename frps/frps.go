package frps

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/connection"
	"log"
	"sync"
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
func (s *ProxyServer) Handler(ctx *connection.Context) {
	req := ctx.GetRequest()
	//conn := ctx.GetConn()
	if ctx.IsHeartBeat() {
		log.Printf("ProxyName [%s], get heartbeat\n", req.ProxyName)
		resp := connection.NewHeartbeatResponse(req.ProxyName)
		ctx.SetResponse(resp)
	} else if ctx.IsEstablishConnection() {

	} else {

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
		go conn.ProcessOutsideRequest(s.Handler)
	}
}
