package frpc

import (
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/connection"
	"github.com/obgnail/go-frp/utils"
)

type ProxyClient struct {
	Name       string
	LocalPort  int64
	RemoteAddr string
	RemotePort int64

	connChan chan *connection.Conn
}

func NewProxyClient(name string, localPort int64, remoteAddr string, remotePort int64) (*ProxyClient, error) {
	tcpConn, err := utils.ConnectServer(remoteAddr, remotePort)
	if err != nil {
		return nil, errors.Trace(err)
	}
	conn := connection.NewConn(tcpConn)

	pc := &ProxyClient{
		Name:       name,
		LocalPort:  localPort,
		RemoteAddr: remoteAddr,
		RemotePort: remotePort,
		connChan:   make(chan *connection.Conn),
	}
	pc.connChan <- conn
	return pc, nil
}

func (c *ProxyClient) Handler(ctx connection.Context) {

}

func (c *ProxyClient) Run() {
	for {
		conn, ok := <-c.connChan
		if !ok {

		}
	}
}
