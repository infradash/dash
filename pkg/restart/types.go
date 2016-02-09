package restart

import (
	"github.com/conductant/gohm/pkg/encoding"
	"time"
)

var (
	DefaultWait          = encoding.Duration{5 * time.Second}
	DefaultRestartConfig = RestartConfig{
		ControllerPathFormat:  "/{{.Domain}}/{{.Controller}}/{{.Version}}/container/{{.Image}}",
		MemberWatchPathFormat: "/{{.Domain}}/{{.Service}}/{{.Version}}/container/{{.Image}}",
		ProxyWatchPathFormat:  "/{{.Domain}}/{{.Service}}/live/watch",
		RestartWaitDuration:   DefaultWait,
	}
)

type RestartConfig struct {
	ProxyUrl              string            `json:"proxy_url,omitempty"`
	ControllerPathFormat  string            `json:"controller_path_format,omitempty"`
	MemberWatchPathFormat string            `json:"member_watch_path_format,omitempty"`
	ProxyWatchPathFormat  string            `json:"proxy_watch_path_format,omitempty"`
	RestartWaitDuration   encoding.Duration `json:"wait,omitempty"`
}
