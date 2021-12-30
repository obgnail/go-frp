package context

type Context interface {
	Process()
}

type EstablishConnectionContext struct {
	req  *EstablishConnectionRequest
	resp *EstablishConnectionResponse
}

func (c *EstablishConnectionContext) Process() {

}
