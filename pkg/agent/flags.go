package agent

import (
	"flag"
)

func (this *Agent) BindFlags() {
	flag.BoolVar(&this.selfRegister, "self_register", true, "Registers self with the registry.")
	flag.IntVar(&this.ListenPort, "port", 25657, "Listening port for agent")
	flag.StringVar(&this.StatusPubsubTopic, "status_topic", "", "Status pubsub topic")

	flag.BoolVar(&this.EnableUI, "enable_ui", false, "Enables UI")
	flag.IntVar(&this.DockerUIPort, "dockerui_port", 25658, "Listening port for dockerui")
	flag.StringVar(&this.UiDocRoot, "ui_docroot", "", "UI DocRoot")
}
