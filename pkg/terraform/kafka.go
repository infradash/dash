package terraform

import (
	"github.com/golang/glog"
	gotemplate "text/template"
)

func (zk *KafkaConfig) Validate() error {
	glog.Infoln("Kafka - validating config")
	c := Config(*zk)
	return c.Validate()
}

func (zk *KafkaConfig) Execute(authToken string, context interface{}, funcs gotemplate.FuncMap) error {
	glog.Infoln("Kafka - executing config")
	c := Config(*zk)
	return c.Execute(authToken, context, funcs)
}

func (this *Terraform) StartKafka() error {
	return nil
}

func (this *Terraform) ConfigureKafka() error {
	if this.Kafka == nil {
		return nil
	}
	return this.Kafka.Execute(this.AuthToken, this, this.template_funcs())
}

func (this *Terraform) VerifyKafka() error {
	return nil
}
