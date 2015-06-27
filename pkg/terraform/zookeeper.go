package terraform

import (
	"fmt"
	_ "github.com/golang/glog"
	"strings"
)

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
