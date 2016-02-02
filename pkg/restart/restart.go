package restart

import (
	"github.com/conductant/gohm/pkg/registry"
	"github.com/conductant/gohm/pkg/template"
	"github.com/conductant/gohm/pkg/zk"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/infradash/dash/pkg/executor"
	"golang.org/x/net/context"
	"strconv"
	"time"
)

type Restart struct {
	RegistryReleaseEntry
	ZkSettings
	RestartConfig

	Controller  string        `json:"controller,omitempty"`
	Initializer *ConfigLoader `json:"-"`
}

func mustNot(err error) {
	if err != nil {
		panic(err)
	}
}

func (this *Restart) Run() error {

	if this.Initializer == nil {
		return ErrNoConfig
	}

	// We don't want the application of template to wipe out Domain, Service, etc. variables
	// So escape them.
	this.Initializer.Context = EscapeVars(ConfigVariables...)
	this.RestartConfig = DefaultRestartConfig

	loaded := false
	var err error
	for {
		loaded, err = this.Initializer.Load(this, "", nil)

		if !loaded || err != nil {
			glog.Infoln("Wait then retry:", err)
			time.Sleep(2 * time.Second)

		} else {
			break
		}
	}

	glog.Infoln("Restarting",
		"Domain=", this.Domain, "Service=", this.Service, "Version=", this.Version, "Path=", this.Path,
		"Image=", this.Image, "Build=", this.Build)

	if this.Controller == "" {
		this.Controller = this.Service + "-controller"
	}

	glog.Infoln("ContainerPath=", this.GetContainerPath())
	glog.Infoln("ProxyWatchPath=", this.GetProxyWatchPath())

	// Load the list of controllers
	url := "zk://" + this.Hosts
	ctx := zk.ContextPutTimeout(context.Background(), this.Timeout)

	reg, err := registry.Dial(ctx, url)
	mustNot(err)
	defer reg.Close()

	controllerPaths, err := reg.List(this.GetContainerPath())
	mustNot(err)

	clients := []*executor.Client{}
	for _, controllerPath := range controllerPaths {
		glog.Infoln("Found controller path", controllerPath)
		if read, _, err := reg.Get(controllerPath); err == nil {
			host, ps := ParseHostPort(string(read))
			port, err := strconv.Atoi(ps)
			mustNot(err)
			client := executor.NewClient().SetHost(host).SetPort(port)
			clients = append(clients, client)
		}
	}

	if len(clients) != len(controllerPaths) {
		panic("Cannot access all controllers")
	}
	glog.Infoln("Clients", clients)
	for _, c := range clients {
		c.SetProxyUrl(this.ProxyUrl)
		info, err := c.GetInfo()
		glog.Infoln("Info=", info, "err=", err)
	}
	return nil
}

func (this *Restart) GetContainerPath() registry.Path {
	applied, err := template.Apply([]byte(this.ContainerPathFormat), this)
	mustNot(err)
	return registry.NewPath(string(applied))
}

func (this *Restart) GetProxyWatchPath() registry.Path {
	applied, err := template.Apply([]byte(this.ProxyWatchPathFormat), this)
	mustNot(err)
	return registry.NewPath(string(applied))
}
