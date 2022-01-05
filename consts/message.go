package consts

import "encoding/json"

type MessageType string

const (
	TypeUnknown MessageType = "1000"

	TypeServerInit          MessageType = "1001"
	TypeServerWaitHeartbeat MessageType = "1002"

	TypeProxyServerWaitUserConnect MessageType = "1003"
	TypeProxyServerWaitProxyClient MessageType = "1004"
	TypeProxyServerWorking         MessageType = "1005"

	TypeClientInit                 MessageType = "1006"
	TypeClientWaitHeartbeat        MessageType = "1007"
	TypeProxyClientWaitProxyServer MessageType = "1008"

	TypeProxyClientWorking MessageType = "1009"
)

type Message struct {
	Type      MessageType            `json:"type"`
	Content   string                 `json:"content"`
	ProxyName string                 `json:"proxy_name"`
	Meta      map[string]interface{} `json:"mate"`
}

func NewMessage(typ MessageType, msg string, proxyName string, meta map[string]interface{}) *Message {
	return &Message{Type: typ, Content: msg, ProxyName: proxyName, Meta: meta}
}

func UnmarshalMsg(msgBytes []byte) (msg *Message, err error) {
	msg = &Message{}
	if err = json.Unmarshal(msgBytes, msg); err != nil {
		return
	}
	return
}
