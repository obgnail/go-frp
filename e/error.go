package e

import (
	"fmt"
)

// string
const (
	ModelMessage  = "Message"
	ModelConn     = "Conn"
	ModelApp      = "App"
	ModelClient   = "Client"
	ModelServer   = "Server"
	ModelListener = "Listener"
)

// ErrReason
const (
	ReasonNotFound  = "NotFound"
	ReasonUnmarshal = "Unmarshal"
	ReasonMarshal   = "Marshal"
	ReasonEmpty     = "Empty"
	ReasonClose     = "Close"
	ReasonSend      = "Send"
	ReasonJoin      = "Join"
	ReasonPassword  = "Password"
)

// ErrArg
const (
	App    = "App"
	Server = "Server"
	Client = "Client"

	Listener = "Listener"
	Conn     = "Conn"

	Meta      = "Meta"
	Type      = "type"
	Message   = "Message"
	Heartbeat = "Heartbeat"
)

func New(errModel string, errReason string, args interface{}) error {
	return fmt.Errorf("[%s.%s] [%s]", errModel, errReason, args)
}

func NotFoundError(errModel string, args interface{}) error {
	return New(errModel, ReasonNotFound, args)
}

func EmptyError(errModel string, args interface{}) error {
	return New(errModel, ReasonEmpty, args)
}

func ConnCloseError() error {
	return New(ModelConn, ReasonClose, "")
}

func SendMessageError(args interface{}) error {
	return New(ModelMessage, ReasonSend, args)
}

func UnmarshalMessageError(args interface{}) error {
	return New(ModelMessage, ReasonUnmarshal, args)
}

func MarshalMessageError(args interface{}) error {
	return New(ModelMessage, ReasonMarshal, args)
}

func JoinConnError() error {
	return New(ModelConn, ReasonJoin, "")
}

func SendHeartbeatMessageError() error {
	return New(ModelMessage, ReasonSend, Heartbeat)
}

func InvalidPasswordError(args interface{}) error {
	return New(ModelApp, ReasonPassword, args)
}
