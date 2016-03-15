package restart

import (
	"github.com/conductant/gohm/pkg/encoding"
	"time"
)

var (
	DefaultWait          = encoding.Duration{10 * time.Second}
	DefaultRestartConfig = RestartConfig{
		CurrentImagePathFormat: "/{{.Domain}}/{{.Service}}/live",
		ControllerPathFormat:   "/{{.Domain}}/{{.Controller}}/{{.Version}}/container/{{.RunningImage}}",
		MemberWatchPathFormat:  "/{{.Domain}}/{{.Service}}/{{.Version}}/container/{{.RunningImage}}",
		ProxyWatchPathFormat:   "/{{.Domain}}/{{.Service}}/live/watch",
		RestartWaitDuration:    DefaultWait,
	}
)

type RestartConfig struct {
	ProxyUrl               string            `json:"proxy_url,omitempty"`
	Controller             string            `json:"controller,omitempty"`
	CurrentImagePathFormat string            `json:"current_image_path_format,omitempty"`
	ControllerPathFormat   string            `json:"controller_path_format,omitempty"`
	MemberWatchPathFormat  string            `json:"member_watch_path_format,omitempty"`
	ProxyWatchPathFormat   string            `json:"proxy_watch_path_format,omitempty"`
	RestartWaitDuration    encoding.Duration `json:"wait,omitempty"`
}
