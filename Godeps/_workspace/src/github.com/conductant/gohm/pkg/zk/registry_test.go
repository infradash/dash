package zk

import (
	"fmt"
	"github.com/conductant/gohm/pkg/registry"
	"golang.org/x/net/context"
	. "gopkg.in/check.v1"
	"strings"
	"testing"
	"time"
)

var delay = 200 * time.Millisecond

func TestRegistry(t *testing.T) { TestingT(t) }

type TestSuiteRegistry struct{}

var _ = Suite(&TestSuiteRegistry{})

func (suite *TestSuiteRegistry) SetUpSuite(c *C) {
}

func (suite *TestSuiteRegistry) TearDownSuite(c *C) {
}

func (suite *TestSuiteRegistry) TestUsage(c *C) {
	ctx := ContextPutTimeout(context.Background(), 1*time.Minute)
	url := "zk://" + strings.Join(Hosts(), ",")
	zk, err := registry.Dial(ctx, url)
	c.Assert(err, IsNil)
	c.Log(zk)

	p := registry.NewPath("/unit-test/registry/test")
	v := []byte("test")
	_, err = zk.Put(p, v, false)
	c.Assert(err, IsNil)
	read, _, err := zk.Get(p)
	c.Assert(read, DeepEquals, v)

	check := map[registry.Path]int{}
	for i := 0; i < 10; i++ {
		cp := p.Sub(fmt.Sprintf("child-%d", i))
		_, err = zk.Put(cp, []byte{0}, false)
		c.Assert(err, IsNil)
		check[cp] = i
	}

	list, err := zk.List(p)
	c.Assert(err, IsNil)
	c.Log(list)
	c.Assert(len(list), Equals, len(check))
	for _, p := range list {
		_, has := check[p]
		c.Assert(has, Equals, true)
	}

	// delete all children
	for i := 0; i < 10; i++ {
		cp := p.Sub(fmt.Sprintf("child-%d", i))
		err = zk.Delete(cp)
		c.Assert(err, IsNil)
	}
	list, err = zk.List(p)
	c.Assert(err, IsNil)
	c.Assert(len(list), Equals, 0)

	exists, err := zk.Exists(p.Sub("child-0"))
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
}

func (suite *TestSuiteRegistry) TestEphemeral(c *C) {
	ctx := ContextPutTimeout(context.Background(), 1*time.Minute)
	url := "zk://" + strings.Join(Hosts(), ",")
	zk, err := registry.Dial(ctx, url)
	c.Assert(err, IsNil)
	c.Log(zk)

	p := registry.NewPath("/unit-test/registry/ephemeral")
	v := []byte("test")
	_, err = zk.Put(p, v, true)
	c.Assert(err, IsNil)
	read, _, err := zk.Get(p)
	c.Assert(read, DeepEquals, v)
	exists, _ := zk.Exists(p)
	c.Assert(exists, Equals, true)
	// disconnect
	err = zk.Close()
	c.Assert(err, IsNil)

	// reconnect
	zk, err = registry.Dial(ctx, url)
	c.Assert(err, IsNil)
	_, _, err = zk.Get(p)
	c.Assert(err, Equals, ErrNotExist)
	exists, _ = zk.Exists(p)
	c.Assert(exists, Equals, false)
}

func (suite *TestSuiteRegistry) TestVersions(c *C) {
	ctx := ContextPutTimeout(context.Background(), 1*time.Minute)
	url := "zk://" + strings.Join(Hosts(), ",")
	zk, err := registry.Dial(ctx, url)
	c.Assert(err, IsNil)
	c.Log(zk)

	p := registry.NewPath("/unit-test/registry/version")
	v := []byte("test")
	version, err := zk.Put(p, v, true)
	c.Assert(err, IsNil)
	c.Assert(version, Not(Equals), registry.InvalidVersion)

	read, version2, err := zk.Get(p)
	c.Assert(read, DeepEquals, v)
	c.Assert(version, Equals, version2)

	// try to update with version
	version3, err := zk.PutVersion(p, []byte{1}, version2)
	c.Assert(err, IsNil)
	c.Assert(version3 > version2, Equals, true)

	// now try to delete with outdated version number
	err = zk.DeleteVersion(p, version)
	c.Assert(err, Equals, ErrBadVersion)

	// read again
	cv, version4, err := zk.Get(p)
	c.Assert(err, IsNil)
	c.Assert(version4, Equals, version3)
	c.Assert(cv, DeepEquals, []byte{1})

	// delete again
	err = zk.DeleteVersion(p, version4)
	c.Assert(err, IsNil)

	_, _, err = zk.Get(p)
	c.Assert(err, Equals, ErrNotExist)
}

func (suite *TestSuiteRegistry) TestFollow(c *C) {
	ctx := ContextPutTimeout(context.Background(), 1*time.Minute)
	url := "zk://" + strings.Join(Hosts(), ",")
	zk, err := registry.Dial(ctx, url)
	c.Assert(err, IsNil)
	c.Log(zk)

	p := registry.NewPath("/unit-test/registry/follow")

	_, err = zk.Put(p.Sub("1"), []byte(url+p.Sub("2").String()), false)
	c.Assert(err, IsNil)

	_, err = zk.Put(p.Sub("2"), []byte(url+p.Sub("3").String()), false)
	c.Assert(err, IsNil)

	_, err = zk.Put(p.Sub("3"), []byte(url+p.Sub("4").String()), false)
	c.Assert(err, IsNil)

	_, err = zk.Put(p.Sub("4"), []byte("end"), false)
	c.Assert(err, IsNil)

	path, value, version, err := registry.Follow(ctx, zk, p.Sub("1"))
	c.Assert(err, IsNil)
	c.Assert(value, DeepEquals, []byte("end"))
	c.Assert(path.String(), Equals, url+p.Sub("4").String())
	c.Assert(version > 0, Equals, true)
}

func (suite *TestSuiteRegistry) TestTriggerCreate(c *C) {
	ctx := ContextPutTimeout(context.Background(), 1*time.Minute)
	url := "zk://" + strings.Join(Hosts(), ",")
	zk, err := registry.Dial(ctx, url)
	c.Assert(err, IsNil)
	c.Log(zk)

	p := registry.NewPath(fmt.Sprintf("/unit-test/registry/%d/trigger/create", time.Now().Unix()))

	created, stop, err := zk.Trigger(registry.Create{Path: p})
	c.Assert(err, IsNil)

	count := new(int)
	done := make(chan int)
	go func() {
		for {
			select {
			case e := <-created:
				*count++
				c.Log("**** Got event:", e, " count=", *count)
			case <-done:
				break
			}
		}
	}()

	_, err = zk.Put(p, []byte{1}, false)
	c.Assert(err, IsNil)

	time.Sleep(delay)

	done <- 1

	c.Assert(*count, Equals, 1)
	stop <- 1
}

func (suite *TestSuiteRegistry) TestTriggerDelete(c *C) {
	ctx := ContextPutTimeout(context.Background(), 1*time.Minute)
	url := "zk://" + strings.Join(Hosts(), ",")
	zk, err := registry.Dial(ctx, url)
	c.Assert(err, IsNil)
	c.Log(zk)

	p := registry.NewPath(fmt.Sprintf("/unit-test/registry/%d/trigger/delete", time.Now().Unix()))

	deleted, stop, err := zk.Trigger(registry.Delete{Path: p})
	c.Assert(err, IsNil)

	count := new(int)
	go func() {
		e, open := <-deleted
		if !open {
			return
		}
		*count++
		c.Log("**** Got event:", e, " count=", *count)
	}()

	_, err = zk.Put(p, []byte{1}, false)
	c.Assert(err, IsNil)

	time.Sleep(delay)

	err = zk.Delete(p)
	c.Assert(err, IsNil)

	time.Sleep(delay)

	stop <- 1
	c.Assert(*count, Equals, 1)
}

func (suite *TestSuiteRegistry) TestTriggerChange(c *C) {
	ctx := ContextPutTimeout(context.Background(), 1*time.Minute)
	url := "zk://" + strings.Join(Hosts(), ",")
	zk, err := registry.Dial(ctx, url)
	c.Assert(err, IsNil)
	c.Log(zk)

	p := registry.NewPath(fmt.Sprintf("/unit-test/registry/%d/trigger/change", time.Now().Unix()))

	changed, stop, err := zk.Trigger(registry.Change{Path: p})
	c.Assert(err, IsNil)

	count := new(int)
	done := make(chan int)
	go func() {
		for {
			select {
			case e := <-changed:
				*count++
				c.Log("**** Got event:", e, " count=", *count)
			case <-done:
				break
			}
		}
	}()

	_, err = zk.Put(p, []byte{1}, false)
	c.Assert(err, IsNil)
	c.Log("*** create")

	time.Sleep(delay)

	_, err = zk.Put(p, []byte{2}, false)
	c.Assert(err, IsNil)
	c.Log("*** change")

	time.Sleep(delay)

	_, err = zk.Put(p, []byte{3}, false)
	c.Assert(err, IsNil)
	c.Log("*** change")

	time.Sleep(delay)

	_, err = zk.Put(p, []byte{4}, false)
	c.Assert(err, IsNil)
	c.Log("*** change")

	time.Sleep(delay * 2)

	stop <- 1

	time.Sleep(delay * 2)

	done <- 1

	if *count < 3 {
		panic("Should be at least 3 events... sometimes zk will send 4 changes.")
	}
}

func (suite *TestSuiteRegistry) TestTriggerMembers(c *C) {
	ctx := ContextPutTimeout(context.Background(), 1*time.Minute)
	url := "zk://" + strings.Join(Hosts(), ",")
	zk, err := registry.Dial(ctx, url)
	c.Assert(err, IsNil)
	c.Log(zk)

	p := registry.NewPath(fmt.Sprintf("/unit-test/registry/%d/trigger/members", time.Now().Unix()))

	_, err = zk.Put(p, []byte{1}, false)
	c.Assert(err, IsNil)

	time.Sleep(delay)

	members, stop, err := zk.Trigger(registry.Members{Path: p})
	c.Assert(err, IsNil)

	count := new(int)
	go func() {
		for {
			e := <-members
			*count++
			c.Log("**** Got event:", e, " count=", *count)
		}
	}()

	_, err = zk.Put(p.Sub("1"), []byte{1}, false)
	c.Assert(err, IsNil)

	time.Sleep(delay)

	_, err = zk.Put(p.Sub("2"), []byte{1}, false)
	c.Assert(err, IsNil)

	time.Sleep(delay)

	_, err = zk.Put(p.Sub("3"), []byte{1}, false)
	c.Assert(err, IsNil)

	time.Sleep(delay)

	err = zk.Delete(p.Sub("3"))
	c.Assert(err, IsNil)

	time.Sleep(delay * 2)
	stop <- 1
	c.Assert(*count, Equals, 4)
}
