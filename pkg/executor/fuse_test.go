package executor

import (
	"encoding/json"
	"fmt"
	. "gopkg.in/check.v1"
	"os"
	"testing"
)

func TestFuse(t *testing.T) { TestingT(t) }

type TestSuiteFuse struct {
}

var _ = Suite(&TestSuiteFuse{})

func (suite *TestSuiteFuse) SetUpSuite(c *C) {
}

func (suite *TestSuiteFuse) TearDownSuite(c *C) {
}

func (suite *TestSuiteFuse) TestParseFuse(c *C) {
	spec := `
{
  "mount" : "/mnt/dev/fuse/",
  "resource" : "zk:///unit-test/test-fuse/test",
  "perm" : "0774"
}
`
	f := &Fuse{}
	err := json.Unmarshal([]byte(spec), f)
	c.Assert(err, Equals, nil)
	c.Assert(f.Perm, Equals, "0774")

	var perm os.FileMode = 0777
	fmt.Sscanf(f.Perm, "%v", &perm)
	c.Assert(perm, Equals, os.FileMode(0774))
	err = f.IsValid()
	c.Assert(err, Equals, nil)
}

func (suite *TestSuiteFuse) _TestMount(c *C) {
	dir := c.MkDir()
	f := &Fuse{
		Resource:   "zk:///unit-test/test-fuse/test-mount",
		MountPoint: dir,
	}
	c.Log("mountpoint is", dir)

	err := f.IsValid()
	c.Assert(err, Equals, nil)

	conn, err := f.Mount()
	c.Assert(conn, Not(Equals), nil)
	c.Assert(err, Equals, nil)
}
