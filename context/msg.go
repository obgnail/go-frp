package context

type GeneralResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

type EstablishConnectionRequest struct {
	Type      int64  `json:"type"`
	ProxyName string `json:"proxy_name"`
}

type EstablishConnectionResponse struct {
	GeneralResponse
}


