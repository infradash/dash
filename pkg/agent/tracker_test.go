package agent

import (
	"container/heap"
	"github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
	. "gopkg.in/check.v1"
	"testing"
)

func TestTracker(t *testing.T) { TestingT(t) }

type TestSuiteTracker struct {
}

var _ = Suite(&TestSuiteTracker{})

func (suite *TestSuiteTracker) SetUpSuite(c *C) {
}

func (suite *TestSuiteTracker) TearDownSuite(c *C) {
}

func (suite *TestSuiteTracker) TestIsVersionOlderByBuild(c *C) {
	v1 := "infradash/infradash:develop-1000.124"
	v2 := "infradash/infradash:develop-1002.125"
	_, _, b1, err := dash.ParseVersion(v1)
	c.Assert(err, Equals, nil)
	c.Log("build1=", b1)

	_, _, b2, err := dash.ParseVersion(v2)
	c.Assert(err, Equals, nil)
	c.Log("build2=", b2)

	for i, cmp := range []func(string, string) bool{
		CompareImages,
		//		IsVersionOlderByBuild,
		//		IsVersionOlderByDockerRepoTags,
	} {
		c.Log("i=", i, ", cmp=", cmp)

		c.Assert(cmp(v1, v2), Equals, true)
		c.Assert(cmp(v2, v1), Equals, false)

		c.Assert(cmp("foo/bar:v1.0-23", "foo/bar:v1.0-24"), Equals, true)
		c.Assert(cmp("foo/bar:v1.0-24.5", "foo/bar:v1.0-24.6"), Equals, true)
		c.Assert(cmp("foo/bar:v1.0-24.5", "foo/bar:v2.0-24.6"), Equals, true)
		c.Assert(cmp("foo/bar:v1.0-24.5", "foo/barxxx:v2.0-24.6"), Equals, false)
		c.Assert(cmp("foo/bar:v1.0-23", "foo/baz:v1.0-24"), Equals, false)
		c.Assert(cmp("foo/bar:v1.0.23", "foo/baz:v1.0.24"), Equals, false)
		c.Assert(cmp("foo/bar:v1.0.23", "foo/baz:v1.0.22"), Equals, false)
		c.Assert(cmp("foo/bar:master-9996.2864", "foo/bar:master-10002.2867"), Equals, true)
		c.Assert(cmp("foo/bar:master-9996.2864-10", "foo/bar:master-10002.2867-2"), Equals, true)
		c.Assert(cmp("foo/bar:master-9996.2864-1", "foo/bar:master-9996.2864-10"), Equals, true)
		c.Assert(cmp("foo/bar:master-9996.2864-1", "foo/bazz:master-9996.2864-10"), Equals, false)
	}
}

func (suite *TestSuiteTracker) TestContainers(c *C) {
	ch := &MinVersionHeap{
		NewContainerGroup("infradash/infradash:develop-1234.567"),
		NewContainerGroup("infradash/infradash:develop-1234.568"),
		NewContainerGroup("infradash/infradash:develop-1234.566"),
		NewContainerGroup("infradash/infradash:develop-1232.455"),
		NewContainerGroup("infradash/infradash:develop-1249.600"),
	}
	heap.Init(ch)

	c.Log(ch)

	heap.Push(ch, NewContainerGroup("infradash/infradash:develop-1249.601"))

	for ch.Len() > 0 {
		popd := heap.Pop(ch).(*ContainerGroup)
		c.Log("Garbage collect ", *popd)
		if ch.Len() > 0 {
			c.Assert(CompareImages(popd.Image, (*ch)[0].Image), Equals, true)
		}
	}
}

func (suite *TestSuiteTracker) TestContainerTracker(c *C) {

	ct := NewContainerTracker("test")

	f1 := ct.GetFsm("infradash", &docker.Container{Id: "110", Image: "infradash/infradash:develop-1.1"})
	c.Assert(f1.Current().State, Equals, Created)

	f2 := ct.GetFsm("infradash", &docker.Container{Id: "111", Image: "infradash/infradash:develop-1.1"})
	c.Assert(f2.Current().State, Equals, Created)

	f3 := ct.GetFsm("infradash", &docker.Container{Id: "120", Image: "infradash/infradash:develop-1.2"})
	c.Assert(f3.Current().State, Equals, Created)

	f4 := ct.GetFsm("infradash", &docker.Container{Id: "130", Image: "infradash/infradash:develop-1.3"})
	c.Assert(f4.Current().State, Equals, Created)

	f5 := ct.GetFsm("infradash", &docker.Container{Id: "140", Image: "infradash/infradash:develop-1.4"})
	c.Assert(f5.Current().State, Equals, Created)

	f6 := ct.GetFsm("vdp", &docker.Container{Id: "v140", Image: "infradash/vdp:develop-1.4"})
	c.Assert(f6.Current().State, Equals, Created)

	f7 := ct.GetFsm("sidekiq", &docker.Container{Id: "s1", Image: "infradash/infradash:develop-1.4"})
	c.Assert(f7.Current().State, Equals, Created)

	oldest, err := ct.PeekOldest("wrong")
	c.Assert(err, Equals, ErrUnknownService)

	c.Assert(ct.CountVersions("infradash"), Equals, 4)
	oldest, err = ct.PopOldest("infradash")
	c.Assert(err, Equals, nil)
	c.Assert(ct.CountVersions("infradash"), Equals, 3)

	c.Log(oldest)
	for _, f := range oldest.Instances() {
		f.Next(Stopping, "Stopping to garbage collect", nil)
		f.Next(Stopped, "Stopped", nil)
		f.Next(Removed, "Removed", nil)
	}

	c.Assert(f1.Current().State, Equals, Removed)
	c.Assert(f2.Current().State, Equals, Removed)
	c.Assert(f3.Current().State, Not(Equals), Removed)
	c.Assert(f4.Current().State, Not(Equals), Removed)
	c.Assert(f5.Current().State, Not(Equals), Removed)
	c.Assert(f6.Current().State, Not(Equals), Removed)

	// remove
	ct.RemoveFsm("infradash", &docker.Container{Id: "120", Image: "infradash/infradash:develop-1.2"})
	oldest, err = ct.PopOldest("infradash")
	c.Log(oldest)
	c.Assert(oldest.Image, Equals, "infradash/infradash:develop-1.3")

	instances := ct.Instances("infradash", "infradash/infradash:develop-1.4")
	c.Assert(len(instances), Equals, 1)

	instances = ct.Instances("vdp", "infradash/vdp:develop-1.4")
	c.Assert(len(instances), Equals, 1)

	// Note that you can have same image name, but the service is different
	instances = ct.Instances("sidekiq", "infradash/infradash:develop-1.4")
	c.Assert(len(instances), Equals, 1)
}
