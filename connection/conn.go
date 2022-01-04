package connection

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/consts"
	"io"
	"log"
	"net"
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

func (c *Conn) Close() {
	c.closeFlag = true
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
		log.Fatal("message's type is empty")
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
		return
	}

	if message.Type == "" {
		log.Fatal("[ERROR] message type is nil")
	}
	return
}
