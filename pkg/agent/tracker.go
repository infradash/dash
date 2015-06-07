package agent

import (
	"container/heap"
	"fmt"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/docker"
	"sync"
)

type container_event struct {
	service   ServiceKey
	state     ContainerState
	container *docker.Container
}

type ContainerTracker struct {
	Domain string

	minVersionHeap   map[ServiceKey]*MinVersionHeap
	minStartTimeHeap map[ServiceKey]*MinStartTimeHeap
	statesListener   map[ServiceKey][]chan<- HostContainerStates

	lock sync.Mutex
}

func NewContainerTracker(domain string) *ContainerTracker {
	c := &ContainerTracker{
		Domain:           domain,
		minVersionHeap:   make(map[ServiceKey]*MinVersionHeap),
		minStartTimeHeap: make(map[ServiceKey]*MinStartTimeHeap),
		statesListener:   make(map[ServiceKey][]chan<- HostContainerStates),
	}
	return c
}

type HostContainerStatesChanged <-chan HostContainerStates

func (this *ContainerTracker) AddStatesListener(service ServiceKey) HostContainerStatesChanged {
	if _, has := this.statesListener[service]; !has {
		this.statesListener[service] = []chan<- HostContainerStates{}
	}
	channel := make(chan HostContainerStates)
	this.statesListener[service] = append(this.statesListener[service], channel)
	return channel
}

func (this *ContainerTracker) GetFsm(service ServiceKey, c *docker.Container) *Fsm {
	if ch, has := this.minVersionHeap[service]; has {
		return ch.GetFsm(c)
	} else {
		ch = &MinVersionHeap{}
		heap.Init(ch)
		heap.Push(ch, NewContainerGroup(c.Image))
		this.minVersionHeap[service] = ch
		return ch.GetFsm(c)
	}
}

func (this *ContainerTracker) RemoveFsm(service ServiceKey, c *docker.Container) {
	if ch, has := this.minVersionHeap[service]; has {
		ch.RemoveFsm(c)
		return
	}
}

func (this *ContainerTracker) AddTrackStartTime(service ServiceKey, c *docker.Container) {
	if _, has := this.minStartTimeHeap[service]; !has {
		ch := &MinStartTimeHeap{}
		ch.Init()
		this.minStartTimeHeap[service] = ch
	}
	this.minStartTimeHeap[service].Add(c)
}

func (this *ContainerTracker) RemoveTrackStartTime(service ServiceKey, c *docker.Container) {
	if ch, has := this.minStartTimeHeap[service]; has {
		if ch.Remove(c) {
			return
		}
	}
	// we may have matched to a wrong service because the container lost its metadata like name, etc.
	// so just brute force remove from all
	for _, ch := range this.minStartTimeHeap {
		if ch.Remove(c) {
			return
		}
	}
}

func (this *ContainerTracker) VisitVersions(visit func(service ServiceKey, cg *ContainerGroup)) {
	for s, h := range this.minVersionHeap {
		h.Visit(func(g *ContainerGroup) {
			visit(s, g)
		})
	}
}

func (this *ContainerTracker) VisitStartTimes(visit func(service ServiceKey, c *docker.Container)) {
	for s, h := range this.minStartTimeHeap {
		h.Visit(func(g *docker.Container) {
			visit(s, g)
		})
	}
}

/// Returns the number of versions of a service
func (this *ContainerTracker) CountVersions(service ServiceKey) int {
	if ch, has := this.minVersionHeap[service]; !has {
		return 0
	} else {
		return ch.Len()
	}
}

/// Returns the number of versions of a service
func (this *ContainerTracker) Instances(service ServiceKey, image string) []*Fsm {
	if ch, has := this.minVersionHeap[service]; has {
		return ch.Instances(image)
	}
	return []*Fsm{}
}

func (this *ContainerTracker) PopOldest(service ServiceKey) (*ContainerGroup, error) {
	if ch, has := this.minVersionHeap[service]; !has {
		return nil, ErrUnknownService
	} else {
		if ch.Len() > 0 {
			return heap.Pop(ch).(*ContainerGroup), nil
		} else {
			return nil, ErrHeapEmpty
		}
	}
}

func (this *ContainerTracker) PeekOldest(service ServiceKey) (*ContainerGroup, error) {
	if ch, has := this.minVersionHeap[service]; !has {
		return nil, ErrUnknownService
	} else {
		if ch.Len() > 0 {
			return (*ch)[0], nil
		} else {
			return nil, ErrHeapEmpty
		}
	}
}

// Returns all containers running on the oldest version of a service
func (this *ContainerTracker) OldestVersion(service ServiceKey) (image string, instances []*Fsm) {
	cg, err := this.PeekOldest(service)
	if err != nil {
		return "", []*Fsm{}
	}
	return cg.Image, cg.Instances()
}

func (this *ContainerTracker) Starting(service ServiceKey, c *docker.Container) {
	this.process(&container_event{service: service, state: Starting, container: c})
}

func (this *ContainerTracker) Running(service ServiceKey, c *docker.Container) {
	this.process(&container_event{service: service, state: Running, container: c})
}

func (this *ContainerTracker) Stopping(service ServiceKey, c *docker.Container) {
	this.process(&container_event{service: service, state: Stopping, container: c})
}

func (this *ContainerTracker) Stopped(service ServiceKey, c *docker.Container) {
	this.process(&container_event{service: service, state: Stopped, container: c})
}

func (this *ContainerTracker) Died(service ServiceKey, c *docker.Container) {
	this.process(&container_event{service: service, state: Failed, container: c})
}

func (this *ContainerTracker) Removed(service ServiceKey, c *docker.Container) {
	this.process(&container_event{service: service, state: Removed, container: c})
}

func (this *ContainerTracker) process(event *container_event) {
	this.lock.Lock()
	defer this.lock.Unlock()

	glog.Infoln("Container tracker - observed event", *event)

	fsm := this.GetFsm(event.service, event.container)
	if fsm == nil {
		glog.Warningln("Error processing event", *event)
		return
	}

	current := fsm.Current().State
	next, err := fsm.Next(event.state, fmt.Sprint("Observe container state=", event.state), nil)
	if err != nil {
		glog.Warningln("Error processing event", *event, "Err=", err, "Current=", current, "Next=", event.state)
		return
	}

	glog.Infoln("Service=", event.service, "Id=", event.container.Id[0:12], "Image=", event.container.Image,
		"State change:", current.String(), "=>", fsm.Current().State.String())

	switch next.State {

	case Removed:
		this.RemoveFsm(event.service, event.container)
		glog.Infoln("Removed: Service=", event.service, "Container=", event.container.Id)

	case Failed, Stopped:
		glog.Infoln("Stopped Service=", event.service, "Container=", event.container.Id,
			"On=", event.container.DockerData.State.FinishedAt)
	}

	switch {
	case next.State == Removed:
		this.RemoveTrackStartTime(event.service, event.container)
	case event.container.DockerData != nil:
		this.AddTrackStartTime(event.service, event.container)
	}

	if l, has := this.minStartTimeHeap[event.service]; has {
		l.Visit(func(c *docker.Container) {
			glog.Infoln("+++++++++++++++", c.Image, c.Id[0:12])
		})
	}

	// if there's a state listener then fire the event
	if sl, has := this.statesListener[event.service]; has {
		for _, c := range sl {
			select {
			case c <- this:
			default:
			}
		}
	}
}
