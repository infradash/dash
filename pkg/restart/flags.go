package restart

import (
	"flag"
)

func (this *Restart) BindFlags() {
	flag.StringVar(&this.Controller, "controller", "",
		"Controller for a service; if not specified, derived to be {{.Service}}-controller")
}
