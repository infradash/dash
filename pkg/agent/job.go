package agent

import (
	"encoding/json"
	"fmt"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/docker"
	"github.com/qorio/maestro/pkg/zk"
	"strings"
)

var (
	containerActionTypeNames = map[ContainerActionType]string{
		Start:  "Start",
		Stop:   "Stop",
		Remove: "Remove",
	}
)

func fetchAuthIdentity(zkc zk.ZK, path string) (*docker.AuthIdentity, error) {
	parse := new(docker.AuthIdentity)
	n, err := zkc.Get(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(n.Value, parse)
	return parse, err
}

// implements GlobalServiceState
func (this *Job) Image() (string, string, string, error) {
	return this.image(this.zk)
}

// Return values:
// 1. Full versioned path pointed by the ReleasePath znode, eg. integration.infradash.com/infradash/release_1
// 2. The parsed version / branch, eg. 'release_1'
// 3. The docker image eg. infradash/infradash:release_1-123
func (this *Job) image(zkc zk.ZK) (string, string, string, error) {
	if this.ImagePath == "" {
		return "", "", "", ErrNoSchedulerReleasePath
	}
	glog.Infoln("Container image from image path", this.ImagePath)
	docker_info, err := zkc.Get(this.ImagePath)
	if err != nil {
		return "", "", "", err
	}
	image := docker_info.GetValueString()
	version := image[strings.LastIndex(image, ":")+1:]
	return fmt.Sprintf("/%s/%s/%s", this.domain, this.service, version), version, image, nil
}

// implements GlobalServiceState
func (this *Job) Instances() (int, error) {
	_, version, image, err := this.Image()
	if err != nil {
		return -1, err
	}

	key, _, err := RegistryKeyValue(KImage, map[string]interface{}{
		"Domain":  this.domain,
		"Service": this.service,
		"Version": version,
		"Image":   image,
	})
	if err != nil {
		return -1, err
	}

	n, err := this.zk.Get(key)
	switch err {
	case nil:
		// Note: this takes a snapshot at this moment in time... It's possible that
		// there are other processes removing or adding children immediately.
		// TODO - need to revisit this static algorithm!!!!
		return int(n.CountChildren()), nil
	case zk.ErrNotExist:
		return 0, nil
	default:
		return -1, err
	}
}

// Defer assignment of container image and container name to external sources.  This for example allow
// us to implement a pull base
func (this *Job) Execute(zkc zk.ZK, dockerc *docker.Docker) error {

	for i, action := range this.Actions {
		opts := action.ContainerControl

		switch action.Action {
		case Start:

			// Get the image of the container
			var pull *docker.Image
			if this.assignImage != nil {
				img, err := this.assignImage(i, &opts)
				if err != nil {
					return err
				}
				pull = img
				opts.Image = img.Repository + ":" + img.Tag
			} else if opts.Image != "" {
				k := strings.LastIndex(opts.Image, ":")
				repository, tag := opts.Image[0:k], opts.Image[k+1:]
				pull = &docker.Image{
					Repository: repository,
					Tag:        tag,
				}
			}

			if pull == nil {
				return ErrNoImage
			}

			// Pull Image -- blocking call
			login := this.AuthIdentity
			if this.DockerAuthInfoPath != "" {
				if l, err := fetchAuthIdentity(zkc, this.DockerAuthInfoPath); err == nil {
					login = l
				} else {
					return err
				}
			}

			if login == nil {
				return ErrNoImageRegistryAuth
			}

			// Get the name of the container
			if this.assignName != nil && action.ContainerNameTemplate != nil {
				if cn := this.assignName(i, *action.ContainerNameTemplate, &opts); cn != "" {
					opts.ContainerName = cn
				}
			}

			glog.Infoln("START (", this.service, ") ===========================================================")
			glog.Infoln("  Login:", login)
			glog.Infoln("  PullImage:", *pull)
			glog.Infoln("  StartContainer: Image=", opts.Image, "ContainerName=", opts.ContainerName)
			glog.Infoln("  StartContainer: ContainerControl=", *opts.Config, "HostConfig=", *opts.HostConfig)

			stopped, err := dockerc.PullImage(login, pull)
			if err == nil {
				// Block until completion
				glog.Infoln("Starting download of", *pull, "with auth", login)
				download_err := <-stopped
				glog.Infoln("Download of image", pull.Repository+":"+pull.Tag, "completed with err=", download_err)
			} else {
				return err
			}

			glog.Infoln("Docker starting container: Name=", opts.ContainerName, "Opts=", opts)

			container, err := dockerc.StartContainer(login, &opts)
			if err != nil {
				// This case is different than the container fails right after start. This is
				// the case where dockerd cannot fork new processes (due to resource limits)
				// or because of container name conflicts.
				ExceptionEvent(err, opts, "Error starting container: Image=", opts.Image)
				return err
			}
			glog.Infoln("Started container", container.Id[0:12], "from", container.Image, ":", *container)

		case Stop:
		case Remove:
		}
	}
	return nil
}
