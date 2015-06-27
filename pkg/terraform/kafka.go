package terraform

import (
	_ "github.com/golang/glog"
)

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
