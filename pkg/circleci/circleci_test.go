package circleci

import (
	"github.com/qorio/maestro/pkg/circleci"
	. "gopkg.in/check.v1"
	"testing"
)

func TestCircleCi(t *testing.T) { TestingT(t) }

type TestSuiteCircleCi struct {
}

var _ = Suite(&TestSuiteCircleCi{})

func (suite *TestSuiteCircleCi) SetUpSuite(c *C) {
}

func (suite *TestSuiteCircleCi) TearDownSuite(c *C) {
}

func (suite *TestSuiteCircleCi) TestCircleCi(c *C) {

	cc := &CircleCi{
		Build: circleci.Build{
			User:         "qorio",
			Project:      "passport",
			ApiToken:     "891d8faf70a1947adf1d6d9b736ca76129e2aa56",
			BuildNum:     103,
			ArtifactsDir: c.MkDir(),
		},
	}

	err := cc.Run()
	c.Assert(err, Equals, nil)
}
