package agent

import (
	"container/heap"
	"github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
	"strconv"
	"strings"
)

type VersionComparator func(imageA, imageB string) bool

var IsVersionOlderByBuild = func(imageA, imageB string) bool {
	if repoA, _, buildA, err := dash.ParseVersion(imageA); err == nil {
		if repoB, _, buildB, err := dash.ParseVersion(imageB); err == nil {
			if repoA != repoB {
				return false
			}

			p1 := strings.Split(buildA, ".")
			p2 := strings.Split(buildB, ".")

			if a, err := strconv.Atoi(p1[0]); err == nil {
				if b, err := strconv.Atoi(p2[0]); err == nil {
					switch {
					case a < b:
						return true
					case a > b:
						return false
					case a == b:
						if len(p1) != len(p2) {
							return len(p1) < len(p2)
						} else if len(p1) > 1 {
							if a, err := strconv.Atoi(p1[1]); err == nil {
								if b, err := strconv.Atoi(p2[1]); err == nil {
									return a < b
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

var IsVersionOlderByDockerRepoTags = func(imageA, imageB string) bool {
	repoA, tagA, err := dash.ParseDockerImage(imageA)
	if err != nil {
		return false
	}
	repoB, tagB, err := dash.ParseDockerImage(imageB)
	if err != nil {
		return false
	}

	if repoA == repoB {
		return tagA <= tagB
	} else {
		return repoA <= repoB
	}
}

// Min-heap of container groups prioritized by the version
type MinVersionHeap []*ContainerGroup

func (h MinVersionHeap) Len() int            { return len(h) }
func (h MinVersionHeap) Less0(i, j int) bool { return IsVersionOlderByBuild(h[i].Image, h[j].Image) }
func (h MinVersionHeap) Less(i, j int) bool {
	return IsVersionOlderByDockerRepoTags(h[i].Image, h[j].Image)
}
func (h MinVersionHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *MinVersionHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(*ContainerGroup))
}
func (h *MinVersionHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (ch *MinVersionHeap) GetFsm(c *docker.Container) *dash.Fsm {
	for _, cg := range *ch {
		if cg.Image == c.Image {
			return cg.GetFsm(c)
		}
	}
	// container's image doesn't match any known. create new
	cg := NewContainerGroup(c.Image)
	heap.Push(ch, cg)
	return cg.GetFsm(c)
}

func (ch *MinVersionHeap) RemoveFsm(c *docker.Container) {
	for i, cg := range *ch {
		if cg.Image == c.Image {
			cg.RemoveFsm(c)
			if cg.Empty() {
				heap.Remove(ch, i)
				return // iteration now invalid
			}
			return
		}
	}
}

func (ch *MinVersionHeap) Visit(visit func(*ContainerGroup)) {
	if visit == nil {
		return
	}
	for _, cg := range *ch {
		visit(cg)
	}
}

func (ch *MinVersionHeap) Instances(image string) []*dash.Fsm {
	for _, cg := range *ch {
		if cg.Image == image {
			return cg.Instances()
		}
	}
	return []*dash.Fsm{}
}
