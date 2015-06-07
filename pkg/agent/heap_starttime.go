package agent

import (
	"container/heap"
	"github.com/qorio/maestro/pkg/docker"
)

// Min-heap of container prioritized by the start time of the container
type MinStartTimeHeap []*docker.Container

func (h MinStartTimeHeap) Len() int { return len(h) }
func (h MinStartTimeHeap) Less(i, j int) bool {
	return h[i].DockerData.State.StartedAt.Before(h[j].DockerData.State.StartedAt)
}
func (h MinStartTimeHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *MinStartTimeHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(*docker.Container))
}
func (h *MinStartTimeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (ch *MinStartTimeHeap) Init() {
	heap.Init(ch)
}

func (ch *MinStartTimeHeap) Add(c *docker.Container) {
	// linear search to make sure we haven't added this already.
	for _, cc := range *ch {
		if cc.Id == c.Id {
			return
		}
	}
	heap.Push(ch, c)
}

func (ch *MinStartTimeHeap) Remove(c *docker.Container) bool {
	for i, cc := range *ch {
		if cc.Id == c.Id {
			heap.Remove(ch, i)
			return true
		}
	}
	return false
}

func (ch *MinStartTimeHeap) Visit(visit func(*docker.Container)) {
	if visit == nil {
		return
	}
	for _, cg := range *ch {
		visit(cg)
	}
}
