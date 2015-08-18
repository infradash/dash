package agent

import (
	"flag"
)

func (this *Agent) BindFlags() {
	flag.IntVar(&this.ListenPort, "port", 25657, "Listening port for agent")
	flag.BoolVar(&this.selfRegister, "self_register", true, "Registers self with the registry.")
	flag.StringVar(&this.UiDocRoot, "ui_docroot", "", "UI DocRoot")
	flag.BoolVar(&this.EnableUI, "enable_ui", false, "Enables UI")
	flag.StringVar(&this.StatusPubsubTopic, "status_topic", "", "Status pubsub topic")
}
