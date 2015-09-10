package agent

import (
	"bytes"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
	"github.com/qorio/maestro/pkg/zk"
	"strings"
	"text/template"
)

var (
	image_counter = map[string]int{}
)

func NoActions() []Job {
	return []Job{}
}

func get_sequence_by_image(image string) int {
	if c, has := image_counter[image]; has {
		image_counter[image] = c + 1
		return c
	} else {
		image_counter[image] = 1
		return 0
	}
}

func AssignContainerNameFromRegistry(global GlobalServiceState, local HostContainerStates,
	domain string, service ServiceKey) AssignContainerName {
	return func(step int, _template string, opts *docker.ContainerControl) string {
		if _, _, image, err := global.Image(); err == nil {

			context := map[string]interface{}{
				"Step":     step,
				"Sequence": get_sequence_by_image(image),
				"Running":  len(local.Instances(service, image)),
				"Domain":   domain,
				"Service":  service,
				"Image":    image,
				"Tag":      "",
			}
			if _, tag, err := ParseDockerImage(image); err == nil {
				context["Tag"] = tag
			}
			if repo, version, build, err := ParseVersion(image); err == nil {
				context["Repo"] = repo
				context["Version"] = version
				context["Build"] = build
			}

			// Apply the template
			if cname, err := template.New(_template).Parse(_template); err == nil {
				var buff bytes.Buffer
				if err = cname.Execute(&buff, context); err == nil {
					return buff.String()
				}
			}
		}
		return ""
	}
}

func AssignContainerImageFromRegistry(global GlobalServiceState, local HostContainerStates,
	domain string, service ServiceKey) AssignContainerImage {
	return func(i int, cc *docker.ContainerControl) (*docker.Image, error) {
		_, _, new_image, err := global.Image()
		if err != nil {
			return nil, err
		}
		if new_image == "" {
			return nil, ErrNoImage
		}

		k := strings.LastIndex(new_image, ":")
		if k < 0 {
			return nil, ErrNoImage
		}
		repository, tag := new_image[0:k], new_image[k+1:]
		return &docker.Image{
			Registry:   "https://index.docker.io",
			Repository: repository,
			Tag:        tag,
		}, nil
	}
}

// Derive the watch container spec based on the scheduler data.
func (this *Scheduler) GetWatchContainerSpec() *WatchContainerSpec {

	if this.Discover == nil {
		this.Discover = &WatchContainerSpec{}
	}
	// TODO - infer this...
	return this.Discover
}

func (this *Scheduler) Run(domain string, service ServiceKey, global GlobalServiceState,
	channel HostContainerStatesChanged, stopper <-chan bool, inbox SchedulerExecutor) error {

	go func() {
		glog.Infoln("Starting scheduler for Service=", service)
		for {
			select {
			case local := <-channel:

				glog.Infoln("ContainerStates changed. Synchronize.")
				err := this.Synchronize(domain, service, local, global, inbox)
				if err != nil {
					glog.Warningln("Error while synchronzing Service=", service, "Err=", err)
				}

			case stop := <-stopper:

				if stop {
					break
				}
			}
		}
		glog.Infoln("Stop: scheduler for Service=", service)
	}()
	return nil
}

func count_failed_containers(local HostContainerStates, service ServiceKey, image string) int {
	all := local.Instances(service, image)
	count := 0
	for _, c := range all {
		if c.Current().State == Failed {
			count += 1
		}
	}
	return count
}

func (this *Scheduler) IsValid() bool {
	implementations := 0
	if this.Constraint != nil {
		implementations += 1
	}
	if this.RunOnce != nil {
		implementations += 1
	}
	return implementations == 1
}

func (this *Scheduler) Synchronize(domain string, service ServiceKey,
	local HostContainerStates, global GlobalServiceState, control SchedulerExecutor) error {

	key, _, image, err := global.Image()
	switch {
	case err == nil:
	case err == zk.ErrNotExist:
		glog.Warningln("No znode for release", "Domain=", domain, "Service=", service)
		return err

	default:
		glog.Warningln("Error trying to determine current version of", service, ": Err=", err)
		return err
	}
	if image == "" {
		glog.Warningln("Service=", service, "Cannot determine container image to use from Path=", key)
		return ErrCannotDetermineContainerImage
	}

	failed := count_failed_containers(local, service, image)
	if failed >= this.MaxAttempts {
		glog.Warningln("Service=", service, "Max attempts exceeded for Image=", image)
		ExceptionEvent(ErrMaxAttemptsExceeded, image, "Container failures exceeded max attempts, Image=", image)
		return ErrMaxAttemptsExceeded
	}

	localInstances := local.Instances(service, image)
	localRunning, globalRunning := 0, 0
	for _, instance := range localInstances {
		switch instance.Current().State {
		case Running:
			localRunning += 1
		case Starting:
			localRunning += 1
		}
	}
	globalRunning, err = global.Instances()
	if err != nil {
		ExceptionEvent(err, global, "Cannot determine global instances")
		return err
	}

	actions := []Job{}
	switch {
	case this.Constraint != nil:
		count, err := this.Constraint.Schedule(localRunning, globalRunning)
		glog.Infoln("Static scheduler:", count, err)
		if err != nil {
			return err
		}
		for i := 0; i < count; i++ {
			actions = append(actions, this.StartOne(domain, service, global, local))
		}
	default:
		// Without specifying any constraints, we just naively start more instances...
		//actions = []Job{this.StartOne(domain, service, global, local)}
	}

	if control != nil {
		control <- actions
	}
	return nil
}

func (this *Scheduler) StartOne(domain string, service ServiceKey,
	global GlobalServiceState, local HostContainerStates) Job {

	sa := this.Job
	sa.domain = domain
	sa.service = service
	sa.assignName = AssignContainerNameFromRegistry(global, local, domain, service)
	sa.assignImage = AssignContainerImageFromRegistry(global, local, domain, service)

	return sa
}
