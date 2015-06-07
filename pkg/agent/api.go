package agent

import (
	"github.com/infradash/dash/pkg/executor"
	"github.com/qorio/maestro/pkg/docker"
	"github.com/qorio/omni/api"
	"net/http"
)

const (
	GetInfo api.ServiceMethod = iota
	GetExecutorConfig
	ListContainers
	WatchContainer
	ConfigureDomain
	ForwardMessage
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

	GetExecutorConfig: api.MethodSpec{
		Doc: `
Returns the configuration for executor
`,
		UrlRoute:     "/v1/executor/{domain}/{service}",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/json"},
		ResponseBody: Types.ExecutorConfig,
	},

	ListContainers: api.MethodSpec{
		Doc: `
List containers
`,
		UrlRoute:     "/v1/containers/{domain}/{service}/",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/json"},
		RequestBody:  Types.Containers,
	},

	WatchContainer: api.MethodSpec{
		Doc: `
Start watching container start/stops
`,
		UrlRoute:     "/v1/watch/container/{domain}/{service}",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json"},
		RequestBody:  Types.WatchContainerSpec,
	},

	ConfigureDomain: api.MethodSpec{
		Doc: `
Configure the domain
`,
		UrlRoute:     "/v1/domain",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json"},
		RequestBody:  Types.DomainConfig,
	},

	ForwardMessage: api.MethodSpec{
		Doc: `
Forwards a message to the target specified in the message
`,
		UrlRoute:     "/v1/forward",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json"},
		RequestBody: func(*http.Request) interface{} {
			return &struct {
				Method  string                  `json:"method"`
				Url     string                  `json:"url"`
				Message *map[string]interface{} `json:"message"`
			}{}
		},
	},
}

var Types = struct {
	Info               func(*http.Request) interface{}
	ReleaseAction      func(*http.Request) interface{}
	WatchContainerSpec func(*http.Request) interface{}
	Containers         func(*http.Request) interface{}
	DomainConfig       func(*http.Request) interface{}
	ExecutorConfig     func(*http.Request) interface{}
}{
	Info:               func(*http.Request) interface{} { return &Info{} },
	WatchContainerSpec: func(*http.Request) interface{} { return &WatchContainerSpec{} },
	Containers:         func(*http.Request) interface{} { return []docker.Container{} },
	DomainConfig:       func(*http.Request) interface{} { return &DomainConfig{} },
	ExecutorConfig:     func(*http.Request) interface{} { return &executor.ExecutorConfig{} },
}
