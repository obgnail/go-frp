package frps

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/connection"
	"github.com/obgnail/go-frp/context"
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

func (p *ProxyServer) Lock() {
	p.mutex.Lock()
}

func (p *ProxyServer) Unlock() {
	p.mutex.Unlock()
}

// 所有连接发送的数据都会到handler函数处理
func (p *ProxyServer) Handler(ctx *context.Context) {

}

func (p *ProxyServer) Process() {
	go func() {
		for {
			conn, err := p.listener.GetConn()
			if err != nil {
				fmt.Println(err)
				continue
			}
			conn.Process(p.Handler)
		}
	}()
}

func (p *ProxyServer) Server() {
	if p == nil {
		err := fmt.Errorf("proxy server is nil")
		log.Fatal(err)
	}
	if p.listener == nil {
		err := fmt.Errorf("proxy server has no listener")
		log.Fatal(err)
	}
	p.Process()
}
