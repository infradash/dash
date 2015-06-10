package agent

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
	"github.com/qorio/maestro/pkg/zk"
	"sync"
	"time"
)

type Domain struct {
	Domain string `json:"domain"`

	RegistryContainerEntry

	Config *DomainConfig

	Identity string `json:"id"`

	zk     zk.ZK          `json:"-"`
	docker *docker.Docker `json:"-"`

	triggers *ZkWatcher

	lock               sync.Mutex
	container_watchers map[string]chan<- bool

	agent   *Agent
	tracker *ContainerTracker

	schedulers       map[ServiceKey]*Scheduler
	scheduleExecutor *ScheduleExecutor
}

func (this *Domain) Register() error {
	attempts := 0
	for {
		err := this.do_register()
		if err != nil {
			if attempts == 12 {
				return err
			} else {
				time.Sleep(5 * time.Second)
				attempts += 1
			}
		} else {
			return nil
		}
	}
}

func (this *Domain) do_register() error {
	if this.zk == nil {
		return ErrNotConnectedToRegistry
	}

	key, value, err := RegistryKeyValue(KDash, this)
	glog.Infoln("Register self as key=", key, "value=", value, "err=", err)
	if err != nil {
		return err
	}
	n, err := this.zk.CreateEphemeral(key, nil)
	if err != nil {
		return err
	} else {
		err = n.Set([]byte(value))
		if err == nil {
			// Update this only on successful registration
			this.Identity = key
		}
	}
	return nil
}

func (this *Domain) StartServices(tags QualifyByTags) (*Domain, error) {
	// Schedulers
	for service, scheduler := range this.Config.Schedulers {
		if scheduler.QualifyByTags.Matches(tags.Tags) {

			applied := new(Scheduler)
			err := ApplyVarSubs(scheduler, applied, MergeMaps(map[string]interface{}{
				"Domain":  this.Domain,
				"Service": service,
			}, EscapeVars(ConfigVariables[2:]...)))

			if err != nil {
				glog.Warningln("Bad spec:", *scheduler)
				return nil, ErrBadSchedulerSpec
			}

			*scheduler = *applied

			if !scheduler.IsValid() {
				glog.Warningln("Bad scheduler specification:", *scheduler)
				ExceptionEvent(ErrBadSchedulerSpec, *scheduler, "Bad scheduler spec")
				return nil, ErrBadSchedulerSpec
			}

			glog.Infoln("Scheduler", "Domain=", this.Domain, "Service=", service, "Scheduler=", *scheduler)
			stop, err := this.AddScheduler(service, scheduler)
			if err != nil {
				ExceptionEvent(err, *scheduler, "Error starting scheduler")
				return nil, err
			} else {
				// TODO
				glog.Infoln("TODO - wire up the stop channel for scheduler:", stop)
			}
		}
	}

	// Vacuums
	for service, vacuumConfig := range this.Config.Vacuums {
		if !vacuumConfig.QualifyByTags.Matches(tags.Tags) {
			continue
		}

		vacuum := NewVacuum(this.Domain, ServiceKey(service), *vacuumConfig, this.tracker, this.docker)
		err := vacuum.Validate()
		if err != nil {
			return nil, err
		}
		err = vacuum.Run()
		if err != nil {
			return nil, err
		}
	}
	return this, nil
}

func (this *Domain) StartScheduleExecutor() (*ScheduleExecutor, error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.scheduleExecutor == nil {
		this.scheduleExecutor = NewScheduleExecutor(this.zk, this.docker)
		err := this.scheduleExecutor.Run()
		if err != nil {
			return nil, err
		}
	}
	return this.scheduleExecutor, nil
}

func (this *Domain) SynchronizeSchedule() error {
	for service, scheduler := range this.schedulers {
		glog.Infoln("Synchronize Service=", service)

		scheduler.Job.zk = this.zk

		global := &scheduler.Job
		local := this.tracker

		err := scheduler.Synchronize(this.Domain, service, local, global, this.scheduleExecutor.Inbox)
		if err != nil {
			glog.Warningln("Error while synchronzing Service=", service, "Err=", err)
		}
	}
	return nil
}

func (this *Domain) AddScheduler(service ServiceKey, scheduler *Scheduler) (chan bool, error) {
	this.schedulers[service] = scheduler
	channel := this.tracker.AddStatesListener(service)
	stopper := make(chan bool)

	scheduler.Job.zk = this.zk
	global := &scheduler.Job

	err := scheduler.Run(this.Domain, service, global, channel, stopper, this.scheduleExecutor.Inbox)
	if err != nil {
		return nil, err
	}

	if scheduler.TriggerPath != nil {
		watch := string(*scheduler.TriggerPath)
		context := &scheduler.Job
		err = this.triggers.AddWatcher(watch, context, func(e zk.Event) bool {

			glog.Infoln("Event for trigger", watch, e)
			if e.State == zk.StateDisconnected {
				glog.Warningln(watch, "disconnected: No action.")
				return true
			}

			syncError := scheduler.Synchronize(this.Domain, service, this.tracker, global, this.scheduleExecutor.Inbox)
			switch syncError {
			case nil:
				return true
			case ErrNoImage:
				ExceptionEvent(syncError, context, "Cannot determine image from referenced node")
				return true
			case zk.ErrNotExist:
				ExceptionEvent(syncError, context, "Referenced node does not exist.  Ok to continue watch")
				return true
			default:
				ExceptionEvent(syncError, context, "Error watching release")
				return true
			}
		})
	}

	return stopper, nil
}

func label(this *docker.Container) string {
	return fmt.Sprintln(this.Image, "@", this.Id[0:12], "(", this.Name, ")")
}

// Based on the scheduler information, derive the rules for discovery and monitoring of containers
func (this *Domain) GetContainerWatcherSpecs() (map[ServiceKey]*WatchContainerSpec, error) {
	matched := map[ServiceKey]*WatchContainerSpec{}
	// Go through all the scheduler settings and derive the WatchContainerSpec
	for service, scheduler := range this.Config.Schedulers {
		if scheduler.QualifyByTags.Matches(this.agent.QualifyByTags.Tags) {
			matched[service] = scheduler.GetWatchContainerSpec()
		}
	}
	return matched, nil
}

func (this *Domain) WatchContainer(service ServiceKey, spec *WatchContainerSpec) error {
	key := fmt.Sprintf("%s-%s", service, spec.Image)

	if this.container_watchers == nil {
		this.container_watchers = make(map[string]chan<- bool)
	}

	this.lock.Lock()
	if _, has := this.container_watchers[key]; !has {

		containerMatcher := new(DiscoveryContainerMatcher).Init()
		specs, err := this.GetContainerWatcherSpecs()
		if err != nil {
			return err
		}

		for svc, spec := range specs {
			containerMatcher.C(this.Domain, svc, spec)
		}

		stop, err := this.docker.WatchContainerMatching(

			containerMatcher.MatcherForDomain(this.Domain, service),

			func(action docker.Action, container *docker.Container) {

				switch action {
				case docker.Create:

					glog.Infoln("#### Container CREATE ####", label(container))
					this.tracker.Starting(service, container)

				case docker.Start:

					glog.Infoln("#### Container START ####", label(container))

					entry, err := BuildRegistryEntry(container, spec.GetMatchContainerPort())

					if err != nil {
						glog.Warningln("Uable to generate registry entries for", *container)
					}

					if entry == nil {
						glog.Warningln("Cannot build registry entry. Not registering:", *container)
					}

					entry.Host = this.Host
					entry.Domain = this.Domain
					entry.Service = string(service)

					err = entry.Register(this.zk)
					k, v, _ := entry.KeyValue()
					if err != nil {
						glog.Warningln("Error registering", k, err)
						return
					} else {
						glog.Infoln("Registered", k, v)
					}

					this.tracker.Running(service, container)

				case docker.Die, docker.Stop, docker.Remove:

					glog.Infoln("#### Container DIE / STOP / REMOVE ####", label(container))

					entry, err := BuildRegistryEntry(container, 0)

					// Update zk
					if err == nil && entry != nil {

						entry.Host = this.Host
						entry.Domain = this.Domain
						entry.Service = string(service)

						err = entry.Remove(this.zk) // blocks
						if err != nil {
							glog.Warningln("Error trying to remove zk entry. Cannot sync state")
						}
					}

					// Update the tracker
					switch action {
					case docker.Die:
						this.tracker.Died(service, container)
					case docker.Stop:
						this.tracker.Stopped(service, container)
					case docker.Remove:
						this.tracker.Removed(service, container)
					}
				}
			})
		if err == nil {
			this.container_watchers[key] = stop
		}
	}
	this.lock.Unlock()
	return nil
}

// Containers in this domain
func (this *Domain) ListContainers(service ServiceKey) ([]*docker.Container, error) {
	// TODO finish this
	return this.docker.ListContainers()
}

func (this *Domain) fetchAuthIdentity(path string) (*docker.AuthIdentity, error) {
	parse := new(docker.AuthIdentity)
	n, err := this.zk.Get(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(n.Value, parse)
	return parse, err
}
