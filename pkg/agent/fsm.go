package agent

import (
	. "github.com/infradash/dash/pkg/dash"
)

const (
	Created ContainerState = iota
	Starting
	Running  // ready running
	Stopping // initiated stop
	Stopped  // stopped
	Failed   // uninitiated / unexpected stop
	Removed  // removed
)

var (
	container_state_labels = map[ContainerState]string{
		Created:  "container:created",
		Starting: "container:starting",
		Running:  "container:running",
		Stopping: "container:stopping",
		Stopped:  "container:stopped",
		Failed:   "container:failed",
		Removed:  "container:removed",
	}

	ContainerFsm = containerFsm{
		Created:  []State{Starting, Running, Failed, Stopped, Removed},
		Starting: []State{Running, Failed, Stopping},
		Running:  []State{Running, Failed, Stopping, Stopped},
		Stopping: []State{Failed, Stopped},
		Stopped:  []State{Removed},
		Failed:   []State{Removed},
	}
)

type ContainerState int
type containerFsm map[ContainerState][]State

func (this ContainerState) String() string {
	return container_state_labels[this]
}

func (this ContainerState) Equals(that State) bool {
	if typed, ok := that.(ContainerState); ok {
		return typed == this
	}
	return false
}

func (this containerFsm) Instance(initial ContainerState) *Fsm {
	return NewFsm(this, initial)
}

func (this containerFsm) Next(s State) (v []State, c bool) {
	if typed, ok := s.(ContainerState); ok {
		v, c = this[typed]
		return
	}
	return nil, false
}
