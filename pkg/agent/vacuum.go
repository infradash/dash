package agent

import (
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
	"time"
)

// Vacuum - Garbage Collector -- watches and removes unwanted stopped containers.

type VacuumStop chan<- bool

type VacuumByVersions struct {
	VersionsToKeep int `json:"versions_to_keep"`
}

type VacuumByStartTime struct {
}

type VacuumConfig struct {
	QualifyByTags

	RemoveImage        bool   `json:"remove_image,omitempty"`
	ExportContainer    bool   `json:"export_container,omitempty"`
	ExportDestination  string `json:"exoprt_destination,omitempty"`
	RunIntervalSeconds uint32 `json:"run_interval_seconds,omitempty"`

	// Option when by version
	ByVersion *VacuumByVersions `json:"by_version,omitempty"`

	// Option when by start time
	ByStartTime *VacuumByStartTime `json:"by_start_time,omitempty"`

	runInterval time.Duration
}

type Vacuum struct {
	Domain  string
	Service ServiceKey
	Config  VacuumConfig
	Stop    VacuumStop

	stop   chan bool
	local  HostContainerStates
	ticker *time.Ticker
	docker *docker.Docker
}

func NewVacuum(domain string, service ServiceKey, config VacuumConfig,
	local HostContainerStates, docker *docker.Docker) *Vacuum {

	stop := make(chan bool)

	if config.RunIntervalSeconds == 0 {
		config.RunIntervalSeconds = 1
	}
	config.runInterval = time.Duration(config.RunIntervalSeconds) * time.Second

	ticker := time.NewTicker(config.runInterval)

	vac := &Vacuum{
		Domain:  domain,
		Service: service,
		Config:  config,
		Stop:    stop,
		stop:    stop,
		local:   local,
		ticker:  ticker,
		docker:  docker,
	}

	return vac
}

func (this *Vacuum) Validate() error {
	if this.Config.ByVersion != nil && this.Config.ByVersion.VersionsToKeep < 0 {
		return ErrBadVacuumConfig
	}
	return nil
}

func (this *Vacuum) Run() error {
	go func() {

		for {
			select {
			case stop := <-this.stop:
				if stop {
					glog.Infoln("Stopping Vacuum:", "Domain=", this.Domain, "Service=", this.Service)
					this.ticker.Stop()
				}
			case <-this.ticker.C:
				err := this.do_vacuum()
				if err != nil {
					glog.Warningln("Vacuum failed:", "Domain=", this.Domain, "Service=", this.Service, "Err=", err)
					ExceptionEvent(err, this, "Vacuum failed")
				}
			}
		}
	}()
	return nil
}

func (this *Vacuum) do_vacuum() error {

	switch {
	case this.Config.ByStartTime != nil:

		// TODO

	case this.Config.ByVersion != nil:

		versions := this.local.CountVersions(this.Service)
		if versions <= this.Config.ByVersion.VersionsToKeep {
			return nil
		}

		image, instances := this.local.OldestVersion(this.Service)
		for _, instance := range instances {

			state := instance.Current().State
			containerId := instance.CustomData.(string)
			glog.Infoln("Domain=", this.Domain, "Service=", this.Service,
				"Id=", containerId[0:12], "State=", state.String(), "Image=", image, "to be vacuummed.")

			switch state {
			case Running:
				glog.Infoln("StopContainer", "Id=", containerId)
				err := this.docker.StopContainer(nil, containerId, 10*time.Second)
				glog.Infoln("StopContainer", "Id=", containerId, "Err=", err)
			case Removed:
				if this.Config.RemoveImage {
					glog.Infoln("Container removed.  Now removing image:", image)
					err := this.docker.RemoveImage(image, true, true)
					glog.Infoln("RemoveImage: err=", err)
				}
			case Stopped, Failed:
				glog.Infoln("RemoveContainer", "Id=", containerId)
				err := this.docker.RemoveContainer(nil, containerId, false, false)
				glog.Infoln("RemoveContainer", "Id=", containerId, "Err=", err)
			}
		}
	default:
	}
	return nil
}
