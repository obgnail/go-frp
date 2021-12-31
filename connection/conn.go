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

	outsideRequestChan  chan *Context
	outsideResponseChan chan *Context
	insideRequestChan   chan *Context
	insideResponseChan  chan *Context
}

func NewConn(tcpConn *net.TCPConn) *Conn {
	c := &Conn{
		TcpConn:             tcpConn,
		closeFlag:           false,
		Reader:              bufio.NewReader(tcpConn),
		outsideRequestChan:  make(chan *Context, 1),
		outsideResponseChan: make(chan *Context, 1),
		insideRequestChan:   make(chan *Context, 1),
		insideResponseChan:  make(chan *Context, 1),
	}
	go c.getOutsideRequest()
	go c.sendOutsideResponse()
	go c.sendInsideRequest()
	go c.getInsideResponse()
	return c
}

// 获取从外部主动发来的数据
func (c *Conn) getOutsideRequest() {
	for {
		req, err := c.ReadRequest()
		if err != nil {
			log.Println("[WARN] get outside request msg err:", errors.Trace(err))
			continue
		}
		ctx := NewContext(req, nil, c)
		c.outsideRequestChan <- ctx
	}
}

// 响应从外部主动发来的数据
func (c *Conn) sendOutsideResponse() {
	for {
		ctx, ok := <-c.outsideResponseChan
		if !ok {
			log.Println("[WARN] send outside response conn had closed")
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

// 主动向外部发送数据后，获取返回数据
func (c *Conn) getInsideResponse() {
	for {
		resp, err := c.ReadResponse()
		if err != nil {
			log.Println("[WARN] get outside response msg err:", errors.Trace(err))
			continue
		}
		ctx := NewContext(nil, resp, c)
		c.insideResponseChan <- ctx
	}
}

// 主动向外部发送数据
func (c *Conn) sendInsideRequest() {
	for {
		ctx, ok := <-c.insideRequestChan
		if !ok {
			log.Println("[WARN] send inside request conn had closed")
			return
		}
		if req := ctx.GetRequest(); req != nil {
			err := c.WriteRequest(req)
			if err != nil {
				log.Println("[WARN] write request err:", errors.Trace(err))
			}
		}
	}
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
		close(c.outsideRequestChan)
		close(c.outsideResponseChan)
		close(c.insideRequestChan)
		close(c.insideResponseChan)
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

// 处理外部主动发来的数据
func (c *Conn) ProcessOutsideRequest(handler func(ctx *Context)) {
	for {
		ctx, ok := <-c.outsideRequestChan
		if !ok {
			log.Println("accept conn had closed")
			return
		}
		handler(ctx)
		c.outsideResponseChan <- ctx
	}
}

// 主动向外部发送数据后，处理对方的响应
func (c *Conn) ProcessInsideRequest(ctx *Context, handler func(ctx *Context)) {
	for {
		c.insideRequestChan <- ctx
		resp, err := c.ReadResponse()
		if err != nil {
			log.Println("[WARN] read response err", errors.Trace(err))
		}
		respCtx := NewContext(ctx.GetRequest(), resp, c)
		handler(respCtx)
		c.insideResponseChan <- respCtx
	}
}
