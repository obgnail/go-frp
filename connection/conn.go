package connection

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/juju/errors"
	"github.com/obgnail/go-frp/context"
)

const (
	BufferRequestStartFlag  = '1'
	BufferResponseStartFlag = '2'
	BufferEndFlag           = '\n'
)

type Conn struct {
	TcpConn   *net.TCPConn
	Reader    *bufio.Reader
	closeFlag bool

	ContextChan chan *context.Context
}

func NewConn(tcpConn *net.TCPConn) *Conn {
	c := &Conn{
		TcpConn:     tcpConn,
		closeFlag:   false,
		Reader:      bufio.NewReader(tcpConn),
		ContextChan: make(chan *context.Context, 1024),
	}
	//go c.accept()
	return c
}

//func (c *Conn) accept() {
//	for {
//		ctx := &context.Context{}
//		req, err := c.Read()
//		if err != nil {
//			err = errors.Trace(err)
//			fmt.Println("[WARN] accept msg err:", err)
//			continue
//		}
//		ctx.SetRequest(req)
//		c.ContextChan <- ctx
//	}
//}

func (c *Conn) read() (buff []byte, err error) {
	buff, err = c.Reader.ReadBytes(BufferEndFlag)
	if err == io.EOF {
		c.closeFlag = true
	}
	return buff, err
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

func (c *Conn) WriteRequest(request *context.Request) (err error) {



	return
}

func (c *Conn) ReadRequest() (request *context.Request, err error) {
	buff, err := c.read()
}

func (c *Conn) Read() (request *context.Request, err error) {
	reqBytes, err := c.read()
	if err != nil {
		err = errors.Trace(err)
		return
	}

	request = &context.Request{}
	if err = json.Unmarshal(reqBytes, request); err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func (c *Conn) Send(req *context.Request) (err error) {
	if req == nil {
		err = fmt.Errorf("has no req")
		return
	}
	ctx := &context.Context{}
	ctx.SetRequest(req)
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

func (c *Conn) Process(handler func(ctx *context.Context)) {
	go func() {
		for {
			ctx, ok := <-c.ContextChan
			if !ok {
				log.Println("Conn had closed")
				return
			}
			handler(ctx)
		}
	}()
}
