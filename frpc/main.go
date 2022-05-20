package main

import (
	"github.com/juju/errors"
	"github.com/obgnail/go-frp/consts"
	log "github.com/sirupsen/logrus"
)

func main() {
	appClientList := []*consts.AppInfo{
		{Name: "SSH", LocalPort: 22, Password: ""},
		{Name: "HTTP", LocalPort: 7777, Password: ""},
	}

	commonProxyClient, err := NewProxyClient("common", 5555, "0.0.0.0", 8888, appClientList)
	if err != nil {
		log.Error(errors.ErrorStack(err))
	}
	commonProxyClient.Run()
}
