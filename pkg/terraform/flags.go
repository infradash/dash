package terraform

import (
	"flag"
	. "github.com/infradash/dash/pkg/dash"
	"os"
)

func (this *Terraform) BindFlags() {
	flag.StringVar(&this.Ip, "ip", os.Getenv(EnvIp), "Host ip")
}
