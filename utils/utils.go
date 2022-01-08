package utils

import (
	"fmt"
	"log"
	"net"
)

func Dail(host string, port int64) (conn *net.TCPConn, err error) {
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return
	}
	conn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		return
	}
	log.Println("[INFO] start connect:", conn.LocalAddr(), "->", conn.RemoteAddr())
	return conn, nil
}
