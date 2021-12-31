package connection

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/juju/errors"
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

	acceptChan chan *Context
	writeChan  chan *Context
}

func NewConn(tcpConn *net.TCPConn) *Conn {
	c := &Conn{
		TcpConn:    tcpConn,
		closeFlag:  false,
		Reader:     bufio.NewReader(tcpConn),
		acceptChan: make(chan *Context, 1),
		writeChan:  make(chan *Context, 1),
	}
	go c.write()
	go c.accept()
	return c
}

func (c *Conn) Close() {
	c.closeFlag = true
}

func (c *Conn) write() {
	for {
		ctx, ok := <-c.writeChan
		if !ok {
			log.Println("[WARN] write conn had closed")
			return
		}

		if resp := ctx.GetResponse(); resp != nil {
			err := c.WriteResponse(resp)
			if err != nil {
				log.Println("[WARN] write response err:", errors.Trace(err))
			}
		}
	}
}

func (c *Conn) accept() {
	for {
		req, err := c.ReadRequest()
		if err != nil {
			log.Println("[WARN] accept request msg err:", errors.Trace(err))
			continue
		}
		ctx := NewContext(req, nil, c)
		c.acceptChan <- ctx
	}
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
		close(c.writeChan)
		close(c.acceptChan)
	}
	return buff, err
}

func (c *Conn) ReadRequest() (request *Request, err error) {
	reqBytes, err := c.Read()
	if err != nil {
		err = errors.Trace(err)
		return
	}
	if err = json.Unmarshal(reqBytes, request); err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *Conn) ReadResponse() (response *Response, err error) {
	respBytes, err := c.Read()
	if err != nil {
		err = errors.Trace(err)
		return
	}
	if err = json.Unmarshal(respBytes, response); err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *Conn) Process(handler func(ctx *Context)) {
	for {
		ctx, ok := <-c.acceptChan
		if !ok {
			log.Println("accept conn had closed")
			return
		}
		handler(ctx)
		c.writeChan <- ctx
	}
}
