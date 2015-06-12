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

type RegistryWatch struct {
	RegistryReleaseEntry

	// If provided, look in here for the actual value instead
	ValueLocation      *RegistryEntryBase  `json:"value_location,omitempty"`
	MatchContainerPort *int                `json:"match_container_port,omitempty"`
	ReloadConfig       *ActionReloadConfig `json:"reload_config,omitempty"`
}

type ActionReloadConfig struct {
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

// TODO - workflow.Task eventually replaces all the other fields.
type ExecutorConfig struct {
	Task          *workflow.Task  `json:"task,omitempty"`
	RegistryKey   string          `json:"registry_key,omitempty"`
	RegistryValue string          `json:"registry_value,omitempty"`
	RegistryWatch []RegistryWatch `json:"registry_watch,omitempty"`
	TailRequest   []TailRequest   `json:"tail_file,omitempty"`
}
