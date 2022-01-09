package main

import (
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/consts"
	log "github.com/sirupsen/logrus"
)

func main() {
	appServerList := []*consts.AppServer{
		{Name: "SSH", ListenPort: 6000, Password: ""},
		{Name: "HTTP", ListenPort: 5000, Password: ""},
	}

	commonProxyServer, err := NewProxyServer("common", "0.0.0.0", 8888, appServerList)
	if err != nil {
		log.Error(errors.ErrorStack(err))
	}
	commonProxyServer.Run()
}
