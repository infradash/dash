package zk

import (
	"github.com/conductant/gohm/pkg/registry"
	"github.com/conductant/gohm/pkg/resource"
	"golang.org/x/net/context"
	. "gopkg.in/check.v1"
	"strings"
	"testing"
	"time"
)

func TestSource(t *testing.T) { TestingT(t) }

type TestSuiteSource struct{}

var _ = Suite(&TestSuiteSource{})

func (suite *TestSuiteSource) SetUpSuite(c *C) {
}

func (suite *TestSuiteSource) TearDownSuite(c *C) {
}

func (suite *TestSuiteSource) TestSourceUsage(c *C) {
	ctx := ContextPutTimeout(context.Background(), 1*time.Minute)
	url := "zk://" + strings.Join(Hosts(), ",")
	zk, err := registry.Dial(ctx, url)
	c.Assert(err, IsNil)
	c.Log(zk)
	defer zk.Close()

	root := registry.NewPathf("/unit-test/zk/%d/source", time.Now().Unix())
	value := []byte("test-value-12345")
	_, err = zk.Put(root, value, false)
	c.Assert(err, IsNil)
	read, _, err := zk.Get(root)
	c.Assert(read, DeepEquals, value)

	sourced, err := resource.Fetch(ctx, url+root.String())
	c.Assert(err, IsNil)
	c.Assert(sourced, DeepEquals, value)

	// Test default to what's in the environment variable -- no hosts specified.
	sourced, err = resource.Fetch(ctx, "zk://"+root.String())
	c.Assert(err, IsNil)
	c.Assert(sourced, DeepEquals, value)

	// Test default to what's in the environment variable -- no hosts specified.
	sourced, err = resource.Fetch(ctx, "zk://bogus/node")
	c.Assert(err, Not(IsNil))
}
