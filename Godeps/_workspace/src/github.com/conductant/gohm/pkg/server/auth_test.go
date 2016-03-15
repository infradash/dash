package server

import (
	. "gopkg.in/check.v1"
	"testing"
)

func TestAuth(t *testing.T) { TestingT(t) }

type TestSuiteAuth struct {
}

var _ = Suite(&TestSuiteAuth{})

func (suite *TestSuiteAuth) SetUpSuite(c *C) {
}

func (suite *TestSuiteAuth) TearDownSuite(c *C) {
}
