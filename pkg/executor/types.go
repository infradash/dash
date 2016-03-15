package executor

import (
	"github.com/qorio/maestro/pkg/pubsub"
	"github.com/qorio/maestro/pkg/registry"
	"github.com/qorio/maestro/pkg/task"
	"github.com/qorio/maestro/pkg/zk"
	"github.com/qorio/omni/version"
)

type Info struct {
	Version       version.Build `json:"version"`
	NowUnix       int64         `json:"now_unix,omitempty"`
	UptimeSeconds float64       `json:"uptime_seconds,omitempty"`
	Executor      *Executor     `json:"executor"`
	Environ       []string      `json:"environ"`
}

type ExecutorConfig struct {
	task.Task

	Envs        []string      `json:"source,omitempty"`
	Mounts      []*Fuse       `json:"mount,omitempty"`
	ConfigFiles []*ConfigFile `json:"config"`
	TailFiles   []*TailFile   `json:"tail,omitempty"`
}

type TailFile struct {
	Path   string       `json:"path,omitempty"`
	Topic  pubsub.Topic `json:"topic,omitempty"`
	Stdout bool         `json:"stdout,omitempty"`
	Stderr bool         `json:"stderr,omitempty"`
}

type ConfigFile struct {
	Init        bool             `json:"init,omitempty"`
	Url         string           `json:"url,omitempty"`
	Path        string           `json:"path,omitempty"`
	Description string           `json:"description,omitempty"`
	Reload      *registry.Change `json:"reload"`
	ReloadCmd   []string         `json:"reload_cmd,omitempty"`
}

type Fuse struct {
	MountPoint string `json:"mount"`
	Resource   string `json:"resource"`
	Perm       string `json:"perm,omitempty"`

	zc zk.ZK
}
