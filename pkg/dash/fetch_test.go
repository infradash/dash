package dash

import (
	"encoding/json"
	"fmt"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFetch(t *testing.T) { TestingT(t) }

type TestSuiteFetch struct {
}

var _ = Suite(&TestSuiteFetch{})

// Database set up for circle_ci:
// psql> create role ubuntu login password 'password';
// psql> create database circle_ci with owner ubuntu encoding 'UTF8';
func (suite *TestSuiteFetch) SetUpSuite(c *C) {
}

func (suite *TestSuiteFetch) TearDownSuite(c *C) {
}

func (suite *TestSuiteFetch) TestFetchAndExecuteTemplate(c *C) {

	list := make([]interface{}, 0)

	for i := 0; i < 10; i++ {
		port := 45167 + i
		hostport := struct {
			Host string
			Port string
		}{
			Host: "ip-10-31-81-235",
			Port: fmt.Sprintf("%d", port),
		}
		c.Log("instance= ", hostport)
		list = append(list, hostport)
	}

	data := make(map[string]interface{})
	data["HostPortList"] = list
	c.Log("Data= ", data)

	url := "http://qorio.github.io/public/nginx/nginx.conf"
	config, err := ExecuteTemplateUrl(nil, url, "", data)

	c.Assert(err, Equals, nil)
	c.Log("config= ", string(config))
}

func (suite *TestSuiteFetch) TestFetchAndExecuteTemplate2(c *C) {

	list := make([]interface{}, 0)

	for i := 0; i < 10; i++ {
		port := 45167 + i
		hostport := struct {
			Host string
			Port string
		}{
			Host: "ip-10-31-81-235",
			Port: fmt.Sprintf("%d", port),
		}
		c.Log("instance= ", hostport)
		list = append(list, hostport)
	}

	data := make(map[string]interface{})
	data["HostPortList"] = list
	c.Log("Data= ", data)

	// content
	content := "Some content"
	content_path := filepath.Join(os.TempDir(), "content.test")
	err := ioutil.WriteFile(content_path, []byte(content), 0777)
	c.Assert(err, Equals, nil)
	content_url := "file://" + content_path

	// Make up a template
	config := fmt.Sprintf(`{ "file" : "{{file "%s"}}", "inline":"{{inline "%s"}}" }`,
		content_url, content_url)

	c.Log("Config:", config)

	// write config to disk
	config_path := filepath.Join(os.TempDir(), "config.test")
	err = ioutil.WriteFile(config_path, []byte(config), 0777)
	c.Assert(err, Equals, nil)

	url := "file://" + config_path
	applied, err := ExecuteTemplateUrl(nil, url, "", data)

	c.Assert(err, Equals, nil)
	c.Log("config= ", string(applied))

	// parse the json and test
	parsed := map[string]string{}

	err = json.Unmarshal(applied, &parsed)
	c.Assert(err, Equals, nil)
	c.Log(parsed)

	c.Assert(parsed["inline"], Equals, content)
	// read the file
	f, err := os.Open(parsed["file"])
	c.Assert(err, Equals, nil)
	buff, err := ioutil.ReadAll(f)
	c.Assert(err, Equals, nil)
	c.Assert(string(buff), Equals, content)
}
