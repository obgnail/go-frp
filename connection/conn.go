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

func (c *Conn) Close() {
	c.closeFlag = true
}

func (c *Conn) GetRemoteAddr() (addr string) {
	return c.TcpConn.RemoteAddr().String()
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
		log.Println("[ERROR] Unmarshal msgBytes Error:", string(msgBytes), "END")
		log.Printf(" %s -> %s\n", c.GetRemoteAddr(), c.GetLocalAddr())
		return
	}

	if message.Type == "" {
		log.Fatal("[ERROR] message type is nil")
	}
	return
}

// will block until connection close
func Join(c1 *Conn, c2 *Conn) {
	log.Printf("join c1: %s -> %s\n", c1.GetRemoteAddr(), c1.GetLocalAddr())
	log.Printf("join c2: %s -> %s\n", c2.GetRemoteAddr(), c2.GetLocalAddr())
	var wait sync.WaitGroup
	pipe := func(to *Conn, from *Conn) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()

		var err error
		_, err = io.Copy(to.TcpConn, from.TcpConn)
		if err != nil {
			log.Printf("join conns error, %v\n", err)
		}
	}

	wait.Add(2)
	go pipe(c1, c2)
	go pipe(c2, c1)
	wait.Wait()
	return
}
