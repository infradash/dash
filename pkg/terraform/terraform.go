package terraform

import (
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/infradash/dash/pkg/executor"
	"io/ioutil"
	"time"
)

type Terraform struct {
	executor.Executor
	TerraformConfig

	Initializer   *ConfigLoader `json:"config_loader"`
	Ip            string        `json:"ip"`
	StartTimeUnix int64
}

func (this *Terraform) Run() error {

	glog.Infoln("ServerIp=", this.Ip)

	if this.Initializer == nil {
		return ErrNoConfig
	}
	this.Initializer.Context = this

	loaded := false
	var err error
	for {
		loaded, err = this.Initializer.Load(this, this.AuthToken, nil)

		if !loaded || err != nil {

			glog.Infoln("Wait then retry")
			time.Sleep(2 * time.Second)

		} else {
			break
		}
	}

	if this.Zookeeper != nil {

		this.Zookeeper.Stop = make(chan bool)

		if this.Zookeeper.Template == "" {
			this.Zookeeper.Template = "func://zk_default_template"
		}
		if this.Zookeeper.Endpoint == "" {
			this.Zookeeper.Endpoint = ZkLocalExhibitorConfigEndpoint
		}
		if this.Zookeeper.CheckStatusEndpoint == "" {
			this.Zookeeper.CheckStatusEndpoint = ZkLocalExhibitorGetConfigEndpoint
		}
	}

	if this.Kafka != nil {

		this.Kafka.Stop = make(chan bool)

		if this.Kafka.Template == "" {
			this.Kafka.Template = "func://kafka_default_template"
		}
	}

	if err := this.TerraformConfig.Validate(); err != nil {
		return err
	}

	if this.Zookeeper != nil {

		myidFile := "/var/zookeeper/myid"
		// check for file
		read, err := ioutil.ReadFile(myidFile)
		glog.Infoln("MyId file = ", string(read), "Err=", err)

		if err != nil {
			return err
		}

		if err := this.Zookeeper.Execute(this.AuthToken, this, this.template_funcs()); err != nil {
			glog.Warningln("Execute zk config, err=", err)
			return err
		}
	}

	if this.Kafka != nil {
		if err := this.Kafka.Execute(this.AuthToken, this, this.template_funcs()); err != nil {
			glog.Warningln("Execute kafka config, err=", err)
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
		"zk_default_template": func() string {
			return DefaultZkExhibitorConfig
		},
		"server_id": func() string {
			return GetServerId(Ip(this.Ip), this.Ensemble)
		},
		"kafka_default_template": func() string {
			return DefaultKafkaProperties
		},
	}
}
