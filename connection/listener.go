package connection

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/e"
	"github.com/obgnail/go-frp/utils"
	log "github.com/sirupsen/logrus"
	"net"
)

type Listener struct {
	addr        net.Addr
	tcpListener *net.TCPListener
	connChan    chan *Conn
	ConnList    *utils.Queue
}

func (l *Listener) Close() {
	for l.ConnList.Len() != 0 {
		if c := l.ConnList.Pop(); c != nil {
			if conn := c.(*Conn); !conn.IsClosed() {
				conn.Close()
			}
		}
	}
	l.tcpListener.Close()
}

func (l *Listener) StartListen() {
	if l.tcpListener == nil {
		log.Fatal(e.NotFoundError(e.ModelListener, e.Listener))
	}
	log.Info("start listen :", l.addr)
	for {
		conn, err := l.tcpListener.AcceptTCP()
		if err != nil {
			continue
		}
		log.Infof("get remote conn: %s -> %s", conn.RemoteAddr(), conn.LocalAddr())
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
		ConnList:    utils.NewQueue(),
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
	l.ConnList.Push(conn)
	return conn, nil
}
