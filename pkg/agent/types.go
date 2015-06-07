package agent

import (
	"github.com/infradash/dash/pkg/executor"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
	"github.com/qorio/maestro/pkg/zk"
	"github.com/qorio/omni/version"
	"sync"
	"time"
)

type ContainerActionType int

const (
	Start ContainerActionType = iota
	Stop
	Remove
)

type Info struct {
	Version version.Build `json:"version"`
	Now     time.Time     `json:"now"`
	Uptime  time.Duration `json:"uptime,omitempty"`
	Agent   *Agent        `json:"agent"`
}

type WatchContainerSpec struct {
	QualifyByTags
	docker.Image
	MatchContainerPort *int    `json:"match_container_port"`
	MatchContainerName *string `json:"match_container_name"`
}

// Configuration for the domain
// Note this is effectively a deployment workflow with state transitions.
// TODO - implement state machine to track this for each service.
type DomainConfig struct {
	RegistryContainerEntry

	WatchContainers map[ServiceKey]*WatchContainerSpec `json:"watch_containers,omitempty"`

	WatchGoLives map[ServiceKey]*GoLiveAction `json:"watch_golives,omitempty"`

	PostGoLives map[ServiceKey]*PostGoLiveAction `json:"post_golives,omitempty"`

	WatchRegistry map[ServiceKey][]executor.RegistryWatch `json:"watch_registry,omitempty"`

	WatchFiles map[ServiceKey][]executor.TailRequest `json:"watch_files,omitempty"`

	Schedulers map[ServiceKey]*Scheduler `json:"schedulers,omitempty"`

	Vacuums map[ServiceKey]*VacuumConfig `json:"vacuums,omitempty"`
}

type Scheduler struct {
	QualifyByTags

	Job

	TriggerPath *Trigger `json:"trigger_path,omitempty"`

	Swarm   *SwarmSchedule   `json:"swarm,omitempty"`
	RunOnce *RunOnceSchedule `json:"run_once,omitemtpy"`

	lock sync.Mutex
}

type Trigger string

type AssignContainerName func(step int, template string, opts *docker.ContainerControl) string
type AssignContainerImage func(step int, opts *docker.ContainerControl) (*docker.Image, error)

type Job struct {

	// Registry path where the image to use is stored.
	ImagePath string `json:"image_path,omitempty"`

	// Max attempts at starting a container -- 0 means no bounds
	MaxAttempts int `json:"max_attempts,omitempty"`

	// Optional - Side effects if run multiple times?
	Idempotent bool `json:"idempotent,omitempty"`

	// Path where Docker auth info can be found in the registry.
	// The value at this path is expected to a json struct for AuthConfiguration
	// http://godoc.org/github.com/fsouza/go-dockerclient#AuthConfiguration
	DockerAuthInfoPath string               `json:"auth_info_path"`
	AuthIdentity       *docker.AuthIdentity `json:"auth"`
	Actions            []ContainerAction    `json:"actions,omitempty"`

	domain  string
	service ServiceKey
	zk      zk.ZK

	assignName  AssignContainerName
	assignImage AssignContainerImage

	// TODO - Add fields here to support implementation of barriers, leader election and global locks required
	// to implement semantics like 'only 1 per cluster' and pre-emption (e.g. A after B)
}

type ContainerAction struct {
	// Template for naming the container. Variables:  Group, Sequence, Domain, Service, Image
	// If not provided, docker naming will be used.
	ContainerNameTemplate *string `json:"container_name_template,omitempty" dash:"template"`

	Action ContainerActionType

	docker.ContainerControl
}

// Static / manual scheduler where the instances counts are specified statically per host
type SwarmSchedule struct {
	MinInstancesPerHost *int `json:"min_instances_per_host,omitempty"`
	MaxInstancesPerHost *int `json:"max_instances_per_host,omitempty"`
	MinInstancesGlobal  *int `json:"min_instances_global,omitempty"`
	MaxInstancesGlobal  *int `json:"max_instances_global,omitempty"`
}

type RunOnceSchedule struct {
	Trigger string `json:"trigger"`
}

type GoLiveAction struct {
	QualifyByTags

	RegistryPath string `json:"registry_path"`
	VerifyLive   func() bool
}

type PostGoLiveAction struct {
	QualifyByTags

	RegistryPath        string               `json:"registry_path"`
	RemoveOldContainers *RemoveOldContainers `json:"remove_old_containers,omitempty"`
}

type RemoveOldContainers struct {
	NumOldVersionsToKeep int `json:"num_keep_old_versions"`
}
