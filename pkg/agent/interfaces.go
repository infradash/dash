package agent

import (
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
)

type SchedulerExecutor chan<- []Task

type HostContainerStates interface {
	Instances(service ServiceKey, image string) []*Fsm
	CountVersions(service ServiceKey) int
	OldestVersion(service ServiceKey) (image string, instances []*Fsm)
	VisitVersions(func(service ServiceKey, cg *ContainerGroup))
	VisitStartTimes(func(service ServiceKey, c *docker.Container))
}

type GlobalServiceState interface {
	Image() (path, version, image string, err error)
	Instances() (count int, err error)
}
