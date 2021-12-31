package connection

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/juju/errors"
	"github.com/obgnail/go-frp/context"
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

func (c *Conn) Write(req *context.Request) (err error) {
	buf, _ := json.Marshal(req)
	buffer := bytes.NewBuffer(buf)
	buffer.WriteByte(BufferEndFlag)
	_, err = c.TcpConn.Write(buffer.Bytes())
	if err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *Conn) ReadLine() (buff []byte, err error) {
	buff, err = c.Reader.ReadBytes(BufferEndFlag)
	if err == io.EOF {
		c.closeFlag = true
	}
	return buff, err
}

func (c *Conn) Read() (response *context.Response, err error) {
	respBytes, err := c.ReadLine()
	if err != nil {
		err = errors.Trace(err)
		return
	}

	response = &context.Response{}
	if err = json.Unmarshal(respBytes, response); err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *Conn) Process(ctx *context.Context) (err error) {
	req := ctx.GetRequest()
	if req == nil {
		err = fmt.Errorf("has no req")
		return
	}
	err = c.Write(req)
	if err != nil {
		err = errors.Trace(err)
		return
	}

	resp, err := c.Read()
	if err != nil {
		err = errors.Trace(err)
		return
	}
	ctx.SetResponse(resp)
	return
}
