package connection

import (
	"bufio"
	"net"
)

type Conn struct {
	TcpConn   *net.TCPConn
	Reader    *bufio.Reader
	closeFlag bool
}
