package proxy

import ()

var (
	DefaultProxyConfig = ProxyConfig{
		Port:            8888,
		AuthScopeGET:    "scope-get",
		AuthScopePOST:   "scope-post",
		BackendProtocol: "http",
	}
)

type ProxyConfig struct {
	Port            int    `json:"listen_port,omitempty"`
	PublicKeyUrl    string `json:"public_key_url,omitempty"`
	AuthScopeGET    string `json:"auth_scope_GET,omitempty"`
	AuthScopePOST   string `json:"auth_scope_POST,omitempty"`
	BackendProtocol string `json:"backend_protocol,omitempty"`
}
