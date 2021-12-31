package context

type MessageType int

const (
	TypeHeartbeat MessageType = iota
	TypeEstablishConnection
)

type Response struct {
	Type      MessageType `json:"type"`
	Code      int64       `json:"code"`
	Msg       string      `json:"msg"`
	ProxyName string      `json:"proxy_name"`
}

type Request struct {
	Type      MessageType `json:"type"`
	Msg       string      `json:"msg"`
	ProxyName string      `json:"proxy_name"`
}

func NewRequest(typ MessageType, msg string, proxyName string) *Request {
	return &Request{Type: typ, Msg: msg, ProxyName: proxyName}
}

func NewEstablishConnectionRequest(proxyName string) *Request {
	return NewRequest(TypeEstablishConnection, "", proxyName)
}

func NewHeartbeatRequest(proxyName string) *Request {
	return NewRequest(TypeHeartbeat, "", proxyName)
}
