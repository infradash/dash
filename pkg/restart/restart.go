package restart

import (
	"fmt"
	"github.com/conductant/gohm/pkg/registry"
	"github.com/conductant/gohm/pkg/template"
	"github.com/conductant/gohm/pkg/zk"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/infradash/dash/pkg/executor"
	"golang.org/x/net/context"
	"path"
	"strconv"
	"strings"
	"time"
)

type Restart struct {
	RegistryReleaseEntry
	ZkSettings
	RestartConfig

	Initializer *ConfigLoader `json:"-"`

	ExecuteForReal bool
	reg            registry.Registry
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

	url := "zk://" + this.Hosts
	ctx := zk.ContextPutTimeout(context.Background(), this.Timeout)
	reg, err := registry.Dial(ctx, url)
	mustNot(err)
	defer reg.Close()

	this.reg = reg

	// We don't want the application of template to wipe out Domain, Service, etc. variables
	// So escape them.
	this.Initializer.Context = EscapeVars(ConfigVariables...)

	if this.Initializer.ConfigUrl != "" {
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
	}

	glog.Infoln("Restarting",
		"Domain=", this.Domain, "Service=", this.Service, "Version=", this.Version, "Path=", this.Path,
		"Image=", this.Image, "Build=", this.Build)

	if this.Controller == "" {
		this.Controller = this.Service + "-controller"
	}

	glog.Infoln("ProxyUrl=", this.ProxyUrl)
	glog.Infoln("ControllerPath=", this.GetControllerPath())
	glog.Infoln("MemberWatchPath=", this.GetMemberWatchPath())
	glog.Infoln("ProxyWatchPath=", this.GetProxyWatchPath())
	glog.Infoln("RestartWaitDuration=", this.RestartConfig.RestartWaitDuration.Duration)

	// Load the list of controllers
	controllerPath := this.GetControllerPath()
	controllerPaths, err := reg.List(controllerPath)
	mustNot(err)

	rollingCount := len(controllerPaths)
	glog.Infoln("Checking on containers at", controllerPath, "count=", rollingCount)

	memberWatchPath := this.GetMemberWatchPath()

	// get a count too
	memberPaths, err := reg.List(memberWatchPath)
	memberCount := len(memberPaths)

	glog.Infoln("Watching members in", memberWatchPath, "count=", memberCount)

	if rollingCount != memberCount {
		// It's possibly a misconfiguration.  We expect the member count to be 1:1 with controller count
		panic(fmt.Errorf("Count mismatch: controllers=", rollingCount, "containers=", memberCount))
	}

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

	// Get the current value of the watch
	proxyWatchPath := this.GetProxyWatchPath()
	watchValueString, watchValueVersion, err := reg.Get(proxyWatchPath)
	if err != nil {
		panic(fmt.Errorf("Cannot get live watch value:%v", err))
	}
	watchValue, err := strconv.Atoi(string(watchValueString))
	if err != nil {
		panic(fmt.Errorf("Cannot get integer value for watch: %s, err=%v", proxyWatchPath, err))
	}

	beginRollingRestart := make(chan int)
	incrementedWatch := make(chan int)
	restartComplete := make(chan int)
	processComplete := make(chan int)
	go func() {
		<-beginRollingRestart
		glog.Infoln("Received signal -- begin rolling restart....")
		for _, c := range clients {
			if this.ProxyUrl != "" {
				c.SetProxyUrl(this.ProxyUrl)
			}
			info, err := c.GetInfo()
			glog.Infoln("Info=", info, "err=", err)

			// Send a kill
			err = c.RemoteKill()
			glog.Infoln("Kill=", err)

			<-incrementedWatch
		}

		// here we poll until the count is restored
		for {
			c, err := reg.List(memberWatchPath)
			if err != nil {
				panic(err)
			}
			// In some cases the node member doesn't get deleted soon enough...
			if len(c) >= len(clients) {
				break
			}
			glog.Infoln("Members count don't match original client count.  Waiting....")
			time.Sleep(this.RestartWaitDuration.Duration)
		}
		restartComplete <- 0
	}()

	// Now we set up the watch
	memberChanges, memberWatchStop, err := reg.Trigger(registry.Members{Path: memberWatchPath})
	mustNot(err)
	go func() {

		version := watchValueVersion

		for itr := 1; ; itr++ {

			select {
			case <-memberChanges:
				glog.Infoln("Received membership change. Incrementing proxy watch:", proxyWatchPath)

				var err error

				newVal := []byte(fmt.Sprintf("%d", watchValue+itr))
				version, err = reg.PutVersion(proxyWatchPath, newVal, version)
				if err != nil {
					panic(fmt.Errorf("Failed to update watch: %s, err=%v", proxyWatchPath, err))
				}

				glog.Infoln("Wait", this.RestartWaitDuration.Duration, "before shutting down next node.")
				time.Sleep(this.RestartWaitDuration.Duration)
				incrementedWatch <- itr

			case <-restartComplete:

				time.Sleep(this.RestartWaitDuration.Duration)

				// Send one more...
				glog.Infoln("Received restartComplete. Incrementing proxy watch:", proxyWatchPath)

				newVal := []byte(fmt.Sprintf("%d", watchValue+itr))
				version, err = reg.PutVersion(proxyWatchPath, newVal, version)
				if err != nil {
					panic(fmt.Errorf("Failed to update watch: %s, err=%v", proxyWatchPath, err))
				}

				glog.Infoln("Restart process completed.")
				processComplete <- 0
				return
			}
		}
	}()

	if this.ExecuteForReal {
		glog.Infoln("Begin rolling restart.")
		beginRollingRestart <- 0
	} else {
		glog.Infoln("Not going to commit.  Use --restart.commit flag to actually begin.")
		return nil
	}

	<-processComplete
	memberWatchStop <- 0

	return nil
}

func (this *Restart) GetControllerPath() registry.Path {
	applied, err := template.Apply([]byte(this.ControllerPathFormat), this)
	mustNot(err)
	return registry.NewPath(string(applied))
}

func (this *Restart) GetMemberWatchPath() registry.Path {
	applied, err := template.Apply([]byte(this.MemberWatchPathFormat), this)
	mustNot(err)
	return registry.NewPath(string(applied))
}

func (this *Restart) GetProxyWatchPath() registry.Path {
	applied, err := template.Apply([]byte(this.ProxyWatchPathFormat), this)
	mustNot(err)
	return registry.NewPath(string(applied))
}

func (this *Restart) RunningImage() string {
	if this.Image == "" {
		// Determine the current image based on zk values
		applied, err := template.Apply([]byte(this.CurrentImagePathFormat), this)
		mustNot(err)
		p := registry.NewPath(string(applied))
		buff, _, err := this.reg.Get(p)
		mustNot(err)
		fullPath := strings.Split(string(buff), ",")[0]
		d, f := path.Split(fullPath)
		this.Image = path.Join(path.Base(d), f)
		glog.Infoln("Getting current image from", p, "value=", this.Image)

	}
	return this.Image
}
