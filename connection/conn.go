package connection

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/juju/errors"
	"io"
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

func (c *Conn) Write(buff []byte) (err error) {
	buffer := bytes.NewBuffer(buff)
	buffer.WriteByte(BufferEndFlag)
	_, err = c.TcpConn.Write(buffer.Bytes())
	if err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *Conn) WriteRequest(request *Request) (err error) {
	reqBytes, _ := json.Marshal(request)
	err = c.Write(reqBytes)
	if err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *Conn) WriteResponse(response *Response) (err error) {
	respBytes, _ := json.Marshal(response)
	err = c.Write(respBytes)
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
	return buff, err
}

func (c *Conn) ReadRequest() (request *Request, err error) {
	reqBytes, err := c.Read()
	if err != nil {
		return
	}
	request = &Request{}
	if err = json.Unmarshal(reqBytes, request); err != nil {
		return
	}
	return
}

func (c *Conn) ReadResponse() (response *Response, err error) {
	respBytes, err := c.Read()
	if err != nil {
		return
	}
	response = &Response{}
	if err = json.Unmarshal(respBytes, response); err != nil {
		return
	}
	return
}
