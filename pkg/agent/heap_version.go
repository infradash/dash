package agent

import (
	"container/heap"
	"github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
	"strconv"
	"strings"
)

type VersionComparator func(imageA, imageB string) bool

func CompareImages(imageA, imageB string) bool {
	repoA, tagA, err := dash.ParseDockerImage(imageA)
	if err != nil {
		return false
	}
	repoB, tagB, err := dash.ParseDockerImage(imageB)
	if err != nil {
		return false
	}

	if repoA != repoB {
		return false
	}

	sep := func(c rune) bool { return c == '.' || c == ',' || c == '-' }
	min := func(a, b int) int {
		if a < b {
			return a
		} else {
			return b
		}
	}

	// compare the tags... we tokenize the tags by delimiters such as . and -
	fieldsA := strings.FieldsFunc(tagA, sep)
	fieldsB := strings.FieldsFunc(tagB, sep)
	for i := 0; i < min(len(fieldsA), len(fieldsB)); i++ {
		a, erra := strconv.Atoi(fieldsA[i])
		b, errb := strconv.Atoi(fieldsB[i])
		switch {
		case erra != nil && errb != nil:
			if fieldsA[i] == fieldsB[i] {
				continue
			} else {
				return fieldsA[i] < fieldsB[i]
			}
		case erra == nil && errb == nil:
			if a == b {
				continue
			} else {
				return a < b
			}
		case erra != nil || errb != nil:
			return false
		}
	}
	return false
}

// Min-heap of container groups prioritized by the version
type MinVersionHeap []*ContainerGroup

func (h MinVersionHeap) Len() int { return len(h) }
func (h MinVersionHeap) Less(i, j int) bool {
	return CompareImages(h[i].Image, h[j].Image)
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
