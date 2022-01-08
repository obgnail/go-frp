package consts

import "encoding/json"

type MessageType string

const (
	TypeServerHeartbeat MessageType = "1001" // 来自commonServer的heartbeat信息
	TypeClientHeartbeat MessageType = "1002" // 来自commonClient的heartbeat信息

	TypeInitApp MessageType = "1003" // client请求server开始监听app端口
	TypeAppMsg  MessageType = "1004" // server告知client,已经开始监听的app信息

	TypeAppWaitJoin MessageType = "1005" // commonServer通知client连接到app端口
	TypeClientJoin  MessageType = "1006" // client连接到app端口
)

type Message struct {
	Type      MessageType `json:"type"`
	Content   string      `json:"content"`
	ProxyName string      `json:"proxy_name"`
	Meta      interface{} `json:"mate"`
}

func NewMessage(typ MessageType, msg string, proxyName string, meta interface{}) *Message {
	return &Message{Type: typ, Content: msg, ProxyName: proxyName, Meta: meta}
}

func UnmarshalMsg(msgBytes []byte) (msg *Message, err error) {
	msg = &Message{}
	if err = json.Unmarshal(msgBytes, msg); err != nil {
		return
	}
	return
}
