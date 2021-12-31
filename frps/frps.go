package frps

import (
	"fmt"
	"github.com/juju/errors"
	"log"
	"sync"

	"github.com/obgnail/go-frp/connection"
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

	listener       *connection.Listener  // accept new connection from remote users
	clientConnChan chan *connection.Conn // get client conns from control goroutine
	mutex          sync.Mutex
}

func NewProxyServer(name, bindAddr string, listenPort int64) (*ProxyServer, error) {
	tcpListener, err := connection.NewTCPListener(bindAddr, listenPort)
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

func (p *ProxyServer) Server() {
	if p == nil {
		err := fmt.Errorf("proxy server is nil")
		log.Fatal(err)
	}
	if p.listener == nil {
		err := fmt.Errorf("proxy server has no listener")
		log.Fatal(err)
	}
	p.listener.StartListen()
}

func (p *ProxyServer) Lock() {
	p.mutex.Lock()
}

func (p *ProxyServer) Unlock() {
	p.mutex.Unlock()
}
