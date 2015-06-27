package terraform

import (
	"fmt"
	_ "github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
)

type Terraform struct {
	Identity
	TerraformConfig

	Initializer   *ConfigLoader `json:"config_loader"`
	Ip            string        `json:"ip"`
	StartTimeUnix int64
}

func (this *Terraform) Run() error {

	if this.Initializer == nil {
		return ErrNoConfig
	}
	this.Initializer.Context = this

	loaded, err := this.Initializer.Load(this, this.AuthToken, nil)
	if err != nil {
		return err
	}
	if !loaded {
		return ErrNotLoaded
	}

	if err := this.TerraformConfig.Validate(); err != nil {
		return err
	}

	if this.Zookeeper != nil {
		if err := this.Zookeeper.Execute(this.AuthToken, this, this.template_funcs()); err != nil {
			return err
		}
	}

	if this.Kafka != nil {
		if err := this.Kafka.Execute(this.AuthToken, this, this.template_funcs()); err != nil {
			return err
		}
	}
	return nil
}

func (this *Terraform) template_funcs() map[string]interface{} {
	return map[string]interface{}{
		"zk_hosts": func() string {
			return GetZkHosts(this.Ensemble)
		},
		"zk_servers_spec": func() string {
			return GetZkServersSpec(Server{Ip: Ip(this.Ip)}, this.Ensemble)
		},
		"server_id": func() string {
			return GetServerId(Ip(this.Ip), this.Ensemble)
		},
	}
}

func GetServerId(self Ip, members []Server) string {
	myid := 0
	for id, s := range members {
		if self == s.Ip {
			myid = id + 1
		}
	}
	return fmt.Sprintf("%d", myid)
}
