package server

import (
	. "gopkg.in/check.v1"
	"reflect"
	"runtime"
	"testing"
)

func TestContext(t *testing.T) { TestingT(t) }

type TestSuiteContext struct {
}

var _ = Suite(&TestSuiteContext{})

func (suite *TestSuiteContext) SetUpSuite(c *C) {
	test_bind.fun = suite.TestApi
}

func (suite *TestSuiteContext) TearDownSuite(c *C) {
}

type b struct {
	fun func(*C)
}

var (
	test_bind b
)

func test_show(c *C, pc uintptr) {
	c.Log("pc=", pc)
	f := runtime.FuncForPC(pc)
	fn, line := f.FileLine(pc)
	c.Log("func=", f.Name(), ",", fn, ",", line)
}

func test_func(c *C) {
	pc, _, _, _ := runtime.Caller(1)
	test_show(c, pc)
	test_show(c, reflect.ValueOf(test_bind.fun).Pointer())
}

func (suite *TestSuiteContext) TestApi(c *C) {
	test_func(c)
}
