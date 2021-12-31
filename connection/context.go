package connection

type Context struct {
	req  *Request
	resp *Response
	conn *Conn
}

func NewContext(req *Request, resp *Response, conn *Conn) *Context {
	return &Context{req: req, resp: resp, conn: conn}
}

func (c *Context) GetRequest() *Request {
	return c.req
}

func (c *Context) GetResponse() *Response {
	return c.resp
}

func (c *Context) SetRequest(req *Request) {
	c.req = req
}

func (c *Context) SetResponse(resp *Response) {
	c.resp = resp
}

func (c *Context) SetConn(conn *Conn) {
	c.conn = conn
}

func (c *Context) GetConn() *Conn {
	return c.conn
}

func (c *Context) whichType(typ MessageType) (ret bool) {
	if c.req != nil && c.req.Type == typ {
		ret = true
	}
	if c.resp != nil && c.resp.Type == typ {
		ret = true
	}
	return ret
}

func (c *Context) IsHeartBeat() bool {
	return c.whichType(TypeHeartbeat)
}

func (c *Context) IsEstablishConnection() bool {
	return c.whichType(TypeEstablishConnection)
}
