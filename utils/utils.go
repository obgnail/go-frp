package utils

import (
	"fmt"
	"net"
)

func ConnectServer(host string, port int64) (c *net.TCPConn, err error) {
	serverAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return
	}
	conn, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {
		return
	}
	return conn, nil
}
