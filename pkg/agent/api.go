package agent

import (
	"github.com/qorio/omni/api"
	"net/http"
)

const (
	GetInfo api.ServiceMethod = iota
	HealthCheck
)

var Methods = api.ServiceMethods{

	GetInfo: api.MethodSpec{
		Doc: `
Returns information about the server.
`,
		UrlRoute:     "/v1/info",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/json"},
		ResponseBody: Types.Info,
	},

	HealthCheck: api.MethodSpec{
		Doc: `
Health check
`,
		UrlRoute:     "/health",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/json"},
		ResponseBody: Types.Health,
	},
}

var Types = struct {
	Info   func(*http.Request) interface{}
	Health func(*http.Request) interface{}
}{
	Info:   func(*http.Request) interface{} { return &Info{} },
	Health: func(*http.Request) interface{} { return &Health{} },
}
