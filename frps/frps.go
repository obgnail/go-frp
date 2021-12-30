package frps

import (
	"container/list"
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

	listener         *connection.Listener  // accept new connection from remote users
	waitUserConnChan chan struct{}         // every time accept a new user conn, put struct{} to the channel
	clientConnChan   chan *connection.Conn // get client conns from control goroutine
	userConnList     *list.List            // store user conns
	mutex            sync.Mutex
}

func NewProxyServer(name, bindAddr string, listenPort int64) (*ProxyServer, error) {
	tcpListener, err := connection.NewTCPListener(bindAddr, listenPort)
	if err != nil {
		return nil, err
	}
	ps := &ProxyServer{
		Name:             name,
		BindAddr:         bindAddr,
		ListenPort:       listenPort,
		Status:           Idle,
		listener:         tcpListener,
		waitUserConnChan: make(chan struct{}),
		clientConnChan:   make(chan *connection.Conn),
		userConnList:     list.New(),
	}
	return ps, nil
}

func (p *ProxyServer) Lock() {
	p.mutex.Lock()
}

func (p *ProxyServer) Unlock() {
	p.mutex.Unlock()
}
