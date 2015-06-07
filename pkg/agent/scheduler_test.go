package agent

import (
	. "gopkg.in/check.v1"
	"math"
	"testing"
)

func TestScheduler(t *testing.T) { TestingT(t) }

type TestSuiteScheduler struct {
}

var _ = Suite(&TestSuiteScheduler{})

func (suite *TestSuiteScheduler) SetUpSuite(c *C) {
}

func (suite *TestSuiteScheduler) TearDownSuite(c *C) {
}

func ref(i int) *int {
	return &i
}

func (suite *TestSuiteScheduler) TestSwarmSchedule(c *C) {
	ss := SwarmSchedule{}
	ss.MaxInstancesGlobal = ref(1)

	gmax, gmin, lmax, lmin, err := ss.check()
	c.Assert(err, Equals, nil)
	c.Assert(gmax, Equals, 1)
	c.Assert(gmin, Equals, 0)
	c.Assert(lmax, Equals, 1)
	c.Assert(lmin, Equals, 0)

	ss.MaxInstancesGlobal = nil
	ss.MinInstancesGlobal = ref(2)

	gmax, gmin, lmax, lmin, err = ss.check()
	c.Assert(err, Equals, nil)
	c.Assert(gmax, Equals, math.MaxInt64)
	c.Assert(gmin, Equals, 2)
	c.Assert(lmax, Equals, math.MaxInt64)
	c.Assert(lmin, Equals, 0)

}
