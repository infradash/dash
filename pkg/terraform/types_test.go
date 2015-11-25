package terraform

import (
	"encoding/json"
	. "gopkg.in/check.v1"
	"testing"
)

func TestTypes(t *testing.T) { TestingT(t) }

type TestSuiteTypes struct {
}

var _ = Suite(&TestSuiteTypes{})

func (suite *TestSuiteTypes) SetUpSuite(c *C) {
}

func (suite *TestSuiteTypes) TearDownSuite(c *C) {
}

func (suite *TestSuiteTypes) TestUnmarshal(c *C) {
	text := `
{
  "ensemble" : [
    {  "ip":"10.40.0.1" },  {  "ip":"10.40.0.2" },  {  "ip":"10.40.0.3" }
  ],
  "zookeeper" : {
    "template" : "http://foo.bar.com/template",
    "endpoint" : "http://requestb.in/19qtk3u1"
  }
}
`
	config := &TerraformConfig{}

	err := json.Unmarshal([]byte(text), config)
	c.Assert(err, Equals, nil)
	c.Assert(config.Validate(), Equals, nil)
	c.Assert(len(config.Ensemble), Equals, 3)
}
