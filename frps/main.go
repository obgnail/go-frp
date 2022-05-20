package main

import (
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/consts"
	log "github.com/sirupsen/logrus"
)

func main() {
	commonProxyServer, err := NewProxyServer(
		"common",
		"0.0.0.0",
		8888,
		[]*consts.AppServerInfo{
			{Name: "SSH", Password: ""},
			{Name: "HTTP", Password: ""},
		},
	)
	if err != nil {
		log.Error(errors.ErrorStack(err))
	}
	commonProxyServer.Serve()
}
