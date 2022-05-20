package connection

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/consts"
	"github.com/obgnail/go-frp/e"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"strings"
	"sync"
)

const (
	BufferEndFlag = '\n'
)

type Conn struct {
	TcpConn   *net.TCPConn
	Reader    *bufio.Reader
	closeFlag bool
}

func NewConn(tcpConn *net.TCPConn) *Conn {
	c := &Conn{
		TcpConn:   tcpConn,
		closeFlag: false,
		Reader:    bufio.NewReader(tcpConn),
	}
	return c
}

func (c *Conn) String() string {
	return fmt.Sprintf("%s <===> %s", c.GetRemoteAddr(), c.GetLocalAddr())
}

func (c *Conn) Close() {
	if c.TcpConn != nil && c.closeFlag == false {
		c.closeFlag = true
		c.TcpConn.Close()
	}
}

func (c *Conn) IsClosed() bool {
	return c.closeFlag
}

func (c *Conn) GetRemoteAddr() (addr string) {
	return c.TcpConn.RemoteAddr().String()
}

func (c *Conn) GetRemoteIP() (ip string) {
	return strings.Split(c.GetRemoteAddr(), ":")[0]
}

func (c *Conn) GetLocalAddr() (addr string) {
	return c.TcpConn.LocalAddr().String()
}

func (c *Conn) Send(buff []byte) (err error) {
	buffer := bytes.NewBuffer(buff)
	buffer.WriteByte(BufferEndFlag)
	_, err = c.TcpConn.Write(buffer.Bytes())
	if err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *Conn) SendMessage(msg *consts.Message) (err error) {
	if msg.Type == "" {
		return e.SendMessageError(msg)
	}
	msgBytes, _ := json.Marshal(msg)
	err = c.Send(msgBytes)
	if err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *Conn) Read() (buff []byte, err error) {
	buff, err = c.Reader.ReadBytes(BufferEndFlag)
	if err == io.EOF {
		c.Close()
	}
	return
}

func (c *Conn) ReadMessage() (message *consts.Message, err error) {
	msgBytes, err := c.Read()
	if err != nil {
		return
	}
	message = &consts.Message{}
	if err = json.Unmarshal(msgBytes, message); err != nil {
		err = e.UnmarshalMessageError(string(msgBytes))
		return
	}

	if message.Type == "" {
		err = e.NotFoundError(e.ModelMessage, e.Type)
		return
	}
	return
}

// will block until connection close
func Join(c1 *Conn, c2 *Conn) {
	var wait sync.WaitGroup
	pipe := func(to *Conn, from *Conn) {
		// 链接断开或发生异常时断开
		defer to.Close()
		defer from.Close()
		defer wait.Done()

		var err error
		_, err = io.Copy(to.TcpConn, from.TcpConn)
		if err != nil {
			log.Warn(e.JoinConnError(), err)
		}
	}
	wait.Add(2)
	go pipe(c1, c2)
	go pipe(c2, c1)
	wait.Wait()
	return
}

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func TryGetFreePort(try int) (int, error) {
	for count := 0; count != try; count++ {
		port, err := GetFreePort()
		if err != nil {
			continue
		}
		return port, nil
	}
	return 0, fmt.Errorf("try too much time")
}

func GetFreePorts(count int) (ports []int, err error) {
	maxFailCount := 5
	failCount := 0
	for failCount != maxFailCount || len(ports) != count {
		port, err := GetFreePort()
		if err != nil {
			failCount++
			continue
		}
		ports = append(ports, port)
	}
	if failCount == 5 {
		return nil, fmt.Errorf("get free port err")
	}
	return ports, nil
}
