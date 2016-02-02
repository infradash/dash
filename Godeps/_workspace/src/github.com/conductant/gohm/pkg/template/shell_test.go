package template

import (
	"golang.org/x/net/context"
	. "gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestShell(t *testing.T) { TestingT(t) }

type TestSuiteShell struct {
}

var _ = Suite(&TestSuiteShell{})

func (suite *TestSuiteShell) SetUpSuite(c *C) {
}

func (suite *TestSuiteShell) TearDownSuite(c *C) {
}

func print(c *C, out io.Reader) {
	bytes, err := ioutil.ReadAll(out)
	c.Log(string(bytes), err)
}

func toString(c *C, out io.Reader) string {
	bytes, err := ioutil.ReadAll(out)
	c.Assert(err, IsNil)
	return string(bytes)
}

func (suite *TestSuiteShell) TestRunShell(c *C) {

	f := ExecuteShell(context.Background())
	shell, ok := f.(func(string) (io.Reader, error))
	c.Assert(ok, Equals, true)

	var stdout io.Reader
	var err error
	_, err = shell("echo '***********'")
	c.Assert(err, IsNil)

	stdout, err = shell("echo foo | sed -e 's/f/g/g'")
	sed := toString(c, stdout)
	c.Log("sed=", sed)
	c.Assert(sed, DeepEquals, "goo\n")

	_, err = shell("echo '***********'")
	c.Assert(err, IsNil)

	stdout, err = shell("echo ${TERM}")
	env := toString(c, stdout)
	c.Log("env=", env)
	c.Assert(strings.Trim(env, " \n"), Equals, os.Getenv("TERM"))

	_, err = shell("echo '***********'")
	c.Assert(err, IsNil)

	stdout, err = shell("ps -ef | wc -l | sed -e 's/ //g'")
	ps := toString(c, stdout)
	c.Log("ps=", ps)
	c.Assert(len(ps), Not(Equals), 0)
}
