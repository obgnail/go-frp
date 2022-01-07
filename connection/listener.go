package connection

import (
	"fmt"
	"github.com/juju/errors"
	"log"
	"net"
)

type Listener struct {
	addr        net.Addr
	tcpListener *net.TCPListener
	connChan    chan *Conn
	closeFlag   bool
}

func (l *Listener) Close() {
	l.closeFlag = true
}

func (l *Listener) StartListen() {
	if l.tcpListener == nil {
		err := fmt.Errorf("has no lisener")
		log.Fatal(err)
	}
	log.Println("[INFO] start listen :", l.addr)
	for {
		conn, err := l.tcpListener.AcceptTCP()
		if err != nil {
			if l.closeFlag {
				return
			}
			continue
		}
		log.Printf("[INFO] get remote conn: %s -> %s\n", conn.RemoteAddr(), conn.LocalAddr())
		c := NewConn(conn)
		l.connChan <- c
	}
}

func NewListener(bindAddr string, bindPort int64) (listener *Listener, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return listener, errors.Trace(err)
	}
	listener = &Listener{
		addr:        tcpListener.Addr(),
		tcpListener: tcpListener,
		connChan:    make(chan *Conn, 1),
		closeFlag:   false,
	}
	go listener.StartListen()
	return listener, nil
}

// wait util get one new connection or listener is closed
// if listener is closed, err returned
func (l *Listener) GetConn() (conn *Conn, err error) {
	var ok bool
	conn, ok = <-l.connChan
	if !ok {
		return conn, fmt.Errorf("channel close")
	}
	return conn, nil
}

