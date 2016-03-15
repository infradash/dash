package zk

import (
	"fmt"
	. "gopkg.in/check.v1"
	"testing"
	"time"
)

func TestNode(t *testing.T) { TestingT(t) }

type TestSuiteNode struct{}

var _ = Suite(&TestSuiteNode{})

func (suite *TestSuiteNode) SetUpSuite(c *C) {
}

func (suite *TestSuiteNode) TearDownSuite(c *C) {
}

func testPath(k string) string {
	return fmt.Sprintf("/unit-test/%d/%s", time.Now().Unix(), k)
}

func (suite *TestSuiteNode) TestSet(c *C) {
	z, _ := Connect(Hosts(), 5*time.Second)

	// Create a node
	p := testPath("node/set")
	n1, err := z.CreateNode(p, []byte{0}, false)
	c.Assert(err, IsNil)

	// read it out
	n2, err := z.GetNode(p)
	c.Assert(err, IsNil)

	// n2 updates
	err = n2.Set([]byte{1})
	c.Assert(err, IsNil)

	c.Assert(n2.Version() > n1.Version(), Equals, true)

	// n1 updates... should fail
	err = n1.Set([]byte{3})
	c.Assert(err, Not(Equals), nil)
	c.Assert(err, Equals, ErrBadVersion)
	c.Log("err=", err)

	z.DeleteNode(p)
}
