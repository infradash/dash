package terraform

import (
	"encoding/json"
	"fmt"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/template"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"
	"testing"
)

var (
	tf_config_path     = os.TempDir() + "/tf_config.template"
	zk_template_path   = os.TempDir() + "/zk_conf.json"
	kf_template_path   = os.TempDir() + "/kf_conf.properties"
	kf_properties_path = os.TempDir() + "/kafka.properties"

	tf_config = fmt.Sprintf(`
{
     "ensemble":[ { "ip":"10.40.0.1" }, { "ip":"10.40.0.2" }, { "ip":"10.40.0.3" } ],
     "zookeeper" : {
         "template":"%s",
         "endpoint":"%s"
     },
     "kafka": {
         "template":"%s",
         "endpoint":"%s"
     }
}
`, "file://"+zk_template_path, "http://requestb.in/19qtk3u1", "file://"+kf_template_path, "file://"+kf_properties_path)
)

const (
	zk_config = `
{
      "zookeeperInstallDirectory":"/usr/local/zookeeper",
      "zookeeperDataDirectory":"/var/zookeeper",
      "zookeeperLogDirectory":"",
      "logIndexDirectory":"",
      "autoManageInstancesSettlingPeriodMs":"180000",
      "autoManageInstancesFixedEnsembleSize":"0",
      "autoManageInstancesApplyAllAtOnce":"1",
      "observerThreshold":"999",
      "serversSpec":"{{ zk_servers_spec }}",
      "javaEnvironment":"",
      "log4jProperties":"",
      "clientPort":"2181",
      "connectPort":"2888",
      "electionPort":"3888",
      "checkMs":"30000",
      "cleanupPeriodMs":"43200000",
      "cleanupMaxFiles":"3",
      "backupPeriodMs":"60000",
      "backupMaxStoreMs":"86400000",
      "autoManageInstances":"0",
      "zooCfgExtra":{
          "syncLimit":"5",
	  "tickTime":"2000",
	  "initLimit":"10"
       },
      "backupExtra":{},
      "serverId":-1
}
`

	kf_properties = `
    broker.id={{server_id}}
    port=9093
    log.dir=/tmp/kafka-logs
    zookeeper.connect={{zk_hosts}}
`
)

func TestTerraform(t *testing.T) { TestingT(t) }

type TestSuiteTerraform struct {
	zk_json_url       string
	kf_properties_url string
	tf_config_url     string
}

var _ = Suite(&TestSuiteTerraform{})

func (suite *TestSuiteTerraform) SetUpSuite(c *C) {
	err := ioutil.WriteFile(zk_template_path, []byte(zk_config), 0777)
	c.Assert(err, Equals, nil)
	suite.zk_json_url = "file://" + zk_template_path

	err = ioutil.WriteFile(kf_template_path, []byte(kf_properties), 0777)
	c.Assert(err, Equals, nil)
	suite.kf_properties_url = "file://" + kf_template_path

	err = ioutil.WriteFile(tf_config_path, []byte(tf_config), 0777)
	c.Assert(err, Equals, nil)
	suite.tf_config_url = "file://" + tf_config_path
}

func (suite *TestSuiteTerraform) TearDownSuite(c *C) {
}

func (suite *TestSuiteTerraform) TestApplyConfigJSONTemplate(c *C) {

	config := &TerraformConfig{
		Ensemble: []Server{
			Server{Ip: "10.40.0.1"},
			Server{Ip: "10.40.0.2"},
			Server{Ip: "10.40.0.3"},
		},
		Zookeeper: &Config{
			Template: Url(suite.zk_json_url),
		},
	}

	{
		t := &Terraform{
			Ip:              "10.40.0.1",
			TerraformConfig: *config,
		}

		j, err := config.Zookeeper.execute_template(t.AuthToken, t, t.template_funcs())
		c.Assert(err, Equals, nil)

		m := map[string]interface{}{}

		err = json.Unmarshal(j, &m)
		c.Assert(err, Equals, nil)
		c.Assert(m["serversSpec"], Equals, "S:1:0.0.0.0,S:2:10.40.0.2,S:3:10.40.0.3")
	}

	{
		t := &Terraform{
			Ip:              "10.40.0.2",
			TerraformConfig: *config,
		}

		j, err := config.Zookeeper.execute_template(t.AuthToken, t, t.template_funcs())
		c.Assert(err, Equals, nil)

		m := map[string]interface{}{}

		err = json.Unmarshal(j, &m)
		c.Assert(err, Equals, nil)
		c.Assert(m["serversSpec"], Equals, "S:1:10.40.0.1,S:2:0.0.0.0,S:3:10.40.0.3")
	}

	{
		t := &Terraform{
			Ip:              "10.40.0.3",
			TerraformConfig: *config,
		}

		j, err := config.Zookeeper.execute_template(t.AuthToken, t, t.template_funcs())
		c.Assert(err, Equals, nil)

		m := map[string]interface{}{}

		err = json.Unmarshal(j, &m)
		c.Assert(err, Equals, nil)
		c.Assert(m["serversSpec"], Equals, "S:1:10.40.0.1,S:2:10.40.0.2,S:3:0.0.0.0")
	}
}

func (suite *TestSuiteTerraform) TestExecute(c *C) {

	t := &Terraform{
		Ip: "10.40.0.1",
		Initializer: &ConfigLoader{
			ConfigUrl: suite.tf_config_url,
		},
	}

	err := t.Run()
	c.Assert(err, Equals, nil)

	properties, _, err := template.FetchUrl(t.Kafka.Endpoint.String(), nil)
	c.Assert(err, Equals, nil)
	c.Log(properties)

	_, err = os.Open(kf_properties_path)
	c.Assert(err, Equals, nil)
}
