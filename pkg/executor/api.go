package executor

import (
	"github.com/qorio/omni/api"
	"net/http"
)

const (
	ApiGetInfo api.ServiceMethod = iota
	ApiQuitQuitQuit
)

var Methods = api.ServiceMethods{

	ApiGetInfo: api.MethodSpec{
		Doc: `
Returns information about the server.
`,
		UrlRoute:     "/v1/info",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/json"},
		ResponseBody: Types.Info,
	},
	ApiQuitQuitQuit: api.MethodSpec{
		Doc: `
Exits the process
`,
		UrlRoute:     "/v1/quitquitquit",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json"},
	},
}

var Types = struct {
	Info func(*http.Request) interface{}
}{
	Info: func(*http.Request) interface{} { return &Info{} },
}
