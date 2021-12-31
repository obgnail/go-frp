package context

type Context struct {
	req  *Request
	resp *Response
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
