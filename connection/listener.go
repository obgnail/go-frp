package connection

import (
	"fmt"
	"net"
)

type Listener struct {
	addr        net.Addr
	tcpListener *net.TCPListener
	connChan    chan *Conn
	closeFlag   bool
}

func NewTCPListener(bindAddr string, bindPort int64) (*Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}
	l := &Listener{
		addr:        tcpAddr,
		tcpListener: listener,
		connChan:    make(chan *Conn),
		closeFlag:   false,
	}
	return l, nil
}
