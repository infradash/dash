package executor

import (
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/pubsub"
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

// TODO - task.Task eventually replaces all the other fields.
type ExecutorConfig struct {
	*task.Task

	TailFiles     []TailFile      `json:"tail,omitempty"`
	RegistryWatch []RegistryWatch `json:"watch,omitempty"`
}

type RegistryWatch struct {
	RegistryReleaseEntry

	// If provided, look in here for the actual value instead
	ValueLocation      *RegistryEntryBase `json:"value_location,omitempty"`
	MatchContainerPort *int               `json:"match_container_port,omitempty"`
	Reload             *Reload            `json:"reload,omitempty"`
}

type TailFile struct {
	Path   string       `json:"path,omitempty"`
	Topic  pubsub.Topic `json:"topic,omitempty"`
	Stdout bool         `json:"stdout,omitempty"`
	Stderr bool         `json:"stderr,omitempty"`
}

type Reload struct {
	Description string `json:"description,omitempty"`

	// Url that serves the template e.g. github pages or S3
	ConfigUrl             string `json:"config_url,omitempty"`
	ConfigDestinationPath string `json:"config_destination,omitempty"`

	Cmd []string `json:"cmd,omitempty"`
}
