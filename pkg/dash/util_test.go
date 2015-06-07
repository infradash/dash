package dash

import (
	. "gopkg.in/check.v1"
	"testing"
)

func TestUtil(t *testing.T) { TestingT(t) }

type TestSuiteUtil struct {
}

var _ = Suite(&TestSuiteUtil{})

func (suite *TestSuiteUtil) SetUpSuite(c *C) {
}

func (suite *TestSuiteUtil) TearDownSuite(c *C) {
}

type test struct {
	Line1 string `json:"line1,omitempty"`
	Line2 string `json:"line2,omitempty"`
	Line3 string `json:"line3,omitempty"`
}

func (suite *TestSuiteUtil) TestUtil(c *C) {

	t1 := &test{
		Line1: "{{.Domain}}-{{.Service}}.{{.Version}}.{{.Build}}",
		Line2: "{{.Sequence}}-{{.Id}}",
		Line3: "line3",
	}

	t2 := new(test)
	err := ApplyVarSubs(t1, t2, MergeMaps(map[string]interface{}{
		"Domain":  "test.infradash.com",
		"Service": "infradash",
	}, EscapeVars("Version", "Build", "Sequence", "Id")))

	c.Assert(err, Equals, nil)
	c.Assert(t2.Line1, Equals, "test.infradash.com-infradash.{{.Version}}.{{.Build}}")
	c.Assert(t2.Line2, Equals, t1.Line2)
	c.Assert(t2.Line3, Equals, t1.Line3)

	t3 := new(test)
	err = ApplyVarSubs(t2, t3, MergeMaps(map[string]interface{}{
		"Version":  "release-1",
		"Build":    34,
		"Sequence": 23,
		"Id":       10,
	}, EscapeVars()))

	c.Assert(err, Equals, nil)
	c.Assert(t3.Line1, Equals, "test.infradash.com-infradash.release-1.34")
	c.Assert(t3.Line2, Equals, "23-10")
	c.Assert(t3.Line3, Equals, t1.Line3)
}
