package agent

import (
	"fmt"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
)

// For all instances of a same container image version.
type ContainerGroup struct {
	Image   string
	FsmById map[string]*Fsm
}

func (c ContainerGroup) String() string {
	return fmt.Sprintf("%s (%d) instances", c.Image, len(c.FsmById))
}

func (c ContainerGroup) Instances() []*Fsm {
	list := make([]*Fsm, 0)
	for _, v := range c.FsmById {
		list = append(list, v)
	}
	return list
}

func NewContainerGroup(image string) *ContainerGroup {
	return &ContainerGroup{
		Image:   image,
		FsmById: make(map[string]*Fsm),
	}
}

func (cg *ContainerGroup) GetFsm(c *docker.Container) *Fsm {
	if fsm, has := cg.FsmById[c.Id]; has {
		return fsm
	} else {
		newFsm := ContainerFsm.Instance(Created)
		newFsm.CustomData = c.Id
		cg.FsmById[c.Id] = newFsm
		return newFsm
	}
}

func (cg *ContainerGroup) RemoveFsm(c *docker.Container) {
	if _, has := cg.FsmById[c.Id]; has {
		delete(cg.FsmById, c.Id)
	}
}

func (cg *ContainerGroup) Empty() bool {
	return len(cg.FsmById) == 0
}
