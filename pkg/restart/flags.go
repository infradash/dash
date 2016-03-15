package restart

import (
	"flag"
)

func (this *Restart) BindFlags() {

	this.RestartConfig = DefaultRestartConfig

	flag.StringVar(&this.Controller, "restart.controller", "",
		"Controller for a service; if not specified, derived to be {{.Service}}-controller")
	flag.StringVar(&this.ProxyUrl, "restart.proxy", "",
		"Proxy url if from outside firewall")
	flag.BoolVar(&this.ExecuteForReal, "restart.commit", false,
		"True to actually execute")
}
