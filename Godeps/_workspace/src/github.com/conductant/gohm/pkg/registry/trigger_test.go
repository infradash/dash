package registry

import (
	. "gopkg.in/check.v1"
	"testing"
)

func TestTrigger(t *testing.T) { TestingT(t) }

type TestSuiteTrigger struct {
}

var _ = Suite(&TestSuiteTrigger{})

func (suite *TestSuiteTrigger) SetUpSuite(c *C) {
}

func (suite *TestSuiteTrigger) TearDownSuite(c *C) {
}

func (suite *TestSuiteTrigger) TestUsage(c *C) {
	create := Create{Path: NewPath("/this/is/a/path")}
	c.Assert(create.Base(), Equals, "path")
	c.Assert(create.Dir(), Equals, NewPath("/this/is/a"))
	c.Assert(create.Dir().String(), Equals, "/this/is/a")

	members := (&Members{Path: NewPath("/path/to/parent")}).SetMin(32)
	c.Assert(members.Path, Equals, NewPath("/path/to/parent"))
	c.Assert(*members.Min, Equals, 32)
}

type TestChange struct {
	Path
	Name string
}

func (suite *TestSuiteTrigger) TestTriggerApi(c *C) {
	// This tests the use of the Trigger marker interface to limit
	// a funciton to only Create, Change, Delete and Members
	// without creating four different versions of the same method.
	//
	test := func(t Trigger) {
		c.Log(t)
		switch t := t.(type) {
		case Create:
			c.Log("Is Create:", t)
		case Change:
			c.Log("Is Change:", t)
		case Delete:
			c.Log("Is Delete:", t)
		case Members:
			c.Log("Is Members:", t)
		default:
			c.Log("Is something else")
		}
	}

	test(Create{Path: NewPath("/this/is/create")})

	// Should not compile
	//test(TestChange{Path: NewPath("/this/is/create")})

}
