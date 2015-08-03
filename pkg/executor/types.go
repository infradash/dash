package executor

import (
	"github.com/qorio/maestro/pkg/pubsub"
	"github.com/qorio/maestro/pkg/registry"
	"github.com/qorio/maestro/pkg/task"
	"github.com/qorio/omni/version"
	"time"
)

type Info struct {
	Version  version.Build `json:"version"`
	Now      time.Time     `json:"now"`
	Uptime   time.Duration `json:"uptime,omitempty"`
	Executor *Executor     `json:"executor"`
}

type ExecutorConfig struct {
	task.Task

	TailFiles   []TailFile   `json:"tail,omitempty"`
	ConfigFiles []ConfigFile `json:"configs"`
}

type TailFile struct {
	Path   string       `json:"path,omitempty"`
	Topic  pubsub.Topic `json:"topic,omitempty"`
	Stdout bool         `json:"stdout,omitempty"`
	Stderr bool         `json:"stderr,omitempty"`
}

type ConfigFile struct {
	Url         string          `json:"url,omitempty"`
	Path        string          `json:"path,omitempty"`
	Description string          `json:"description,omitempty"`
	Reload      registry.Change `json:"reload"`

	ReloadCmd []string `json:"reload_cmd,omitempty"`
}
