package proxy

import (
	"github.com/conductant/gohm/pkg/server"
)

var (
	DefaultProxyConfig = ProxyConfig{
		Port:            8888,
		AuthScope:       server.AuthScopeNone,
		BackendProtocol: "http",
	}
)

type ProxyConfig struct {
	Port            int              `json:"listen_port,omitempty"`
	PublicKeyUrl    string           `json:"public_key_url,omitempty"`
	AuthScope       server.AuthScope `json:"auth_scope,omitempty"`
	BackendProtocol string           `json:"backend_protocol,omitempty"`
}
