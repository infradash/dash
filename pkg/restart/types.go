package restart

import ()

var (
	DefaultRestartConfig = RestartConfig{
		ContainerPathFormat:  "/{{.Domain}}/{{.Controller}}/{{.Version}}/container/{{.Image}}",
		ProxyWatchPathFormat: "/{{.Domain}}/{{.Service}}/live/watch",
	}
)

type RestartConfig struct {
	ProxyUrl             string `json:"proxy_url,omitempty"`
	ContainerPathFormat  string `json:"container_path_format,omitempty"`
	ProxyWatchPathFormat string `json:"proxy_watch_path_format,omitempty"`
}
