package terraform

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"strings"
	gotemplate "text/template"
	"time"
)

const (
	ZkLocalExhibitorConfigEndpoint    = "http://localhost:8080/exhibitor/v1/config/set"
	ZkLocalExhibitorGetConfigEndpoint = "http://localhost:8080/exhibitor/v1/config/get-state"
)

func (zk *ZookeeperConfig) Validate() error {
	glog.Infoln("Zookeeper - validating config")
	c := Config(*zk)
	return c.Validate()
}

func (zk *ZookeeperConfig) Execute(authToken string, context interface{}, funcs gotemplate.FuncMap) error {

	<-zk.CheckReady()

	glog.Infoln("Zookeeper - executing config")
	c := Config(*zk)
	return c.Execute(authToken, context, funcs)
}

func (zk *ZookeeperConfig) CheckReady() chan error {

	ready := make(chan error)
	ticker := time.Tick(2 * time.Second)

	go func() {
		for {
			select {

			case <-ticker:

				glog.Infoln("CheckReady: ", zk.CheckStatusEndpoint)

				client := &http.Client{}
				resp, err := client.Get(zk.CheckStatusEndpoint.String())

				glog.Infoln("CheckReady resp=", resp, "Err=", err)

				if err == nil && resp.StatusCode == http.StatusOK {

					type status_t struct {
						Running bool `json:"running"`
					}

					buff, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						ready <- err
					}

					status := new(status_t)
					err = json.Unmarshal(buff, status)

					glog.Infoln("Status=", string(buff), "err=", err)

					// At this point, ready or not just as long we have a response
					if err == nil {
						glog.Infoln("Got valid response from Exhibitor: server running=", status.Running)
						ready <- err
						break
					} else {
						glog.Infoln("Exhibitor not running. Wait.")
					}
				}
			}
		}
	}()
	return ready
}

func GetZkServersSpec(self Server, members []Server) string {
	list := []string{}
	for id, s := range members {
		serverType := "S"
		if self.Observer {
			serverType = "O"
		}
		host := s.Ip
		if self.Ip == s.Ip {
			host = "0.0.0.0"
		}
		list = append(list, fmt.Sprintf("%s:%d:%s", serverType, id+1, host))
	}
	return strings.Join(list, ",")
}

func GetZkHosts(members []Server) string {
	list := []string{}
	for _, s := range members {
		host := s.Ip
		port := 2181
		if s.Port > 0 {
			port = s.Port
		}
		list = append(list, fmt.Sprintf("%s:%d", host, port))
	}
	return strings.Join(list, ",")
}

func (this *Terraform) StartZookeeper() error {
	return nil
}

func (this *Terraform) ConfigureZookeeper() error {
	if this.Zookeeper == nil {
		return nil
	}
	return this.Zookeeper.Execute(this.AuthToken, this, this.template_funcs())
}

func (this *Terraform) VerifyZookeeper() error {
	return nil
}
