package executor

import (
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/workflow"
	"github.com/qorio/omni/version"
	"time"
)

type Info struct {
	Version  version.Build `json:"version"`
	Now      time.Time     `json:"now"`
	Uptime   time.Duration `json:"uptime,omitempty"`
	Executor *Executor     `json:"executor"`
}

// TODO - workflow.Task eventually replaces all the other fields.
type ExecutorConfig struct {
	*workflow.Task

	TailRequest   []TailRequest   `json:"tail,omitempty"`
	RegistryWatch []RegistryWatch `json:"registry_watch,omitempty"`
}

type RegistryWatch struct {
	RegistryReleaseEntry

	// If provided, look in here for the actual value instead
	ValueLocation      *RegistryEntryBase `json:"value_location,omitempty"`
	MatchContainerPort *int               `json:"match_container_port,omitempty"`
	ReloadConfig       *ReloadConfig      `json:"reload_config,omitempty"`
}

type ReloadConfig struct {
	Description string `json:"description,omitempty"`

	// Url that serves the template e.g. github pages or S3
	ConfigTemplateUrl     string `json:"config_template_url,omitempty"`
	ConfigDestinationPath string `json:"config_destination_path,omitempty"`

	ReloadCmd []string `json:"reload_cmd,omitempty"`
}

type TailRequest struct {
	Path         string `json:"path,omitempty"`
	Output       string `json:"output,omitempty"`
	RegistryPath string `json:"registry_path,omitempty"` // Where to look for actual host:port
	MQTTTopic    string `json:"mqtt_topic,omitempty"`
}
