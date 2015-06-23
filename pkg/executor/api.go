package executor

import (
	"github.com/qorio/omni/api"
	"net/http"
)

const (
	ApiGetInfo api.ServiceMethod = iota
	ApiSaveWatchAction
	ApiGetWatchAction
	ApiTailFile
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

	ApiSaveWatchAction: api.MethodSpec{
		Doc: `
Schedules a watch and performs action on value change
`,
		UrlRoute:     "/v1/watch/live/{domain}/{service}",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json"},
		RequestBody:  Types.RegistryWatch,
	},

	ApiGetWatchAction: api.MethodSpec{
		Doc: `
Returns the current config rule by id
`,
		UrlRoute:     "/v1/watch/live/{domain}/{service}",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/json"},
		ResponseBody: Types.RegistryWatch,
	},

	ApiTailFile: api.MethodSpec{
		Doc: `
Tail a file and direct the output to specified location 'stdout, stderr, or websocket url
`,
		UrlRoute:     "/v1/tail",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json"},
		RequestBody:  Types.TailFile,
	},
}

var Types = struct {
	Info          func(*http.Request) interface{}
	RegistryWatch func(*http.Request) interface{}
	TailFile      func(*http.Request) interface{}
}{
	Info:          func(*http.Request) interface{} { return &Info{} },
	RegistryWatch: func(*http.Request) interface{} { return &RegistryWatch{} },
	TailFile:      func(*http.Request) interface{} { return &TailFile{} },
}
