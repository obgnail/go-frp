package utils

import (
	"fmt"
	"log"
	"net"
)

func ConnectServer(host string, port int64) (conn *net.TCPConn, err error) {
	serverAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return
	}
	conn, err = net.DialTCP("tcp", nil, serverAddr)
	if err != nil {
		return
	}
	log.Println("[INFO] start connect:", conn.LocalAddr(), "->", conn.RemoteAddr())
	return conn, nil
}
