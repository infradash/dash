package agent

import (
	"encoding/json"
	"github.com/qorio/maestro/pkg/docker"
	. "gopkg.in/check.v1"
	"testing"
)

func TestAgent(t *testing.T) { TestingT(t) }

type TestSuiteAgent struct {
}

var _ = Suite(&TestSuiteAgent{})

// Database set up for circle_ci:
// psql> create role ubuntu login password 'password';
// psql> create database circle_ci with owner ubuntu encoding 'UTF8';
func (suite *TestSuiteAgent) SetUpSuite(c *C) {
}

func (suite *TestSuiteAgent) TearDownSuite(c *C) {
}

func (suite *TestSuiteAgent) TestParseAuthIdentity(c *C) {
	auth_json := `
{
  "username" : "joe",
  "password" : "password",
  "email"    : "joe@foo.com"
}
`
	a := &docker.AuthIdentity{}
	err := json.Unmarshal([]byte(auth_json), a)
	c.Assert(err, Equals, nil)

	c.Assert(a.Username, Equals, "joe")
	c.Assert(a.Password, Equals, "password")
	c.Assert(a.Email, Equals, "joe@foo.com")
}

func (suite *TestSuiteAgent) TestParseJobPullImage(c *C) {
	release_action := `
{
  "auth" : {
    "username" : "joe",
    "password" : "password",
    "email"    : "joe@foo.com"
  }
}
`
	ra := &Job{}
	err := json.Unmarshal([]byte(release_action), ra)
	c.Assert(err, Equals, nil)

	c.Assert(ra.AuthIdentity.Username, Equals, "joe")
}

func (suite *TestSuiteAgent) TestParseJobStartContainers(c *C) {
	release_action := `
{
  "auth" : {
    "username" : "joe",
    "password" : "password",
    "email"    : "joe@foo.com"
  },
  "actions" : [
    {
      "name": "infradash",
      "Image" : "infradash/infradash:v01.2-45",
      "AttachStdin" : true,
      "Env" : [ "FOO=foo", "BAR=bar", "BAZ=baz" ],
      "Cmd" : [ "bundle", "exec", "unicorn" ],

      "host_config" : {
        "PublishAllPorts" : true,
        "VolumesFrom" : [ "data_instance", "data_instance2" ],
        "PortBindings" : {
            "3000/tcp" : [{ "HostPort" : "41566" }]
        }
      }
    }
  ]
}
`
	ra := &Job{}
	err := json.Unmarshal([]byte(release_action), ra)
	c.Assert(err, Equals, nil)

	c.Assert(ra.Actions[0].Image, Equals, "infradash/infradash:v01.2-45")
	c.Assert(ra.Actions[0].AttachStdin, Equals, true)
	c.Assert(ra.Actions[0].HostConfig.PublishAllPorts, Equals, true)
	c.Assert(ra.Actions[0].HostConfig.PortBindings["3000/tcp"][0].HostPort, Equals, "41566")
}
