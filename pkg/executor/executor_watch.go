package executor

import (
	"errors"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/zk"
	"io/ioutil"
	"os/exec"
)

func (this *Executor) SaveWatchAction(watch *RegistryWatch) error {
	watch_key, _, err := watch.Path, "", error(nil)
	if watch_key == "" {
		watch_key, _, err = RegistryKeyValue(KLiveWatch, watch)
		if err != nil {
			return err
		}
	}

	glog.Infoln("Watching registry key", watch_key)

	return this.watcher.AddWatcher(watch_key, watch, func(e zk.Event) bool {

		glog.Infoln("Observed event for key", watch_key, e)

		if e.State == zk.StateDisconnected {
			glog.Warningln(watch_key, "disconnected. No action.")
			return true // keep watching
		}

		value_location := watch_key
		// read the value -- do we have an actual value node that we should read from?
		if watch.ValueLocation != nil {
			if watch.ValueLocation.Path != "" {
				value_location = watch.ValueLocation.Path
			} else {
				v, _, err := RegistryKeyValue(KLive, watch.ValueLocation)
				if err != nil {
					glog.Warningln("No value location. Stop watching", *watch)
					this.watcher.StopWatch(watch_key)
					return false
				}
				value_location = v
			}
		}

		glog.Infoln("Getting value from", value_location)
		n, err := this.zk.Get(value_location)
		if err != nil {
			glog.Warningln("Dropping watch of", watch_key, "on error", err)
			return false
		}

		glog.Infoln("Observed change at", watch_key, "location=", value_location, "value=", n.GetValueString())

		containers_path, environments_path := ParseLiveValue(n.GetValueString())

		if watch.Reload != nil {
			glog.Infoln("Reload config using values in", containers_path, "and environment in", environments_path)

			// fetch from containers path
			data, err := this.hostport_list_from_zk(containers_path, watch.MatchContainerPort)
			if err != nil {
				glog.Warningln("Failed to fetch containers from", containers_path, "Continue.")
				return true // Keep watching
			}

			configBuff, err := ExecuteTemplateUrl(this.zk, watch.Reload.ConfigUrl, this.AuthToken, data)
			if err != nil {
				return false
			}
			glog.V(100).Infoln("Config template:", string(configBuff))

			err = ioutil.WriteFile(watch.Reload.ConfigDestinationPath, configBuff, 0777)
			if err != nil {
				glog.Warningln("Cannot write config to", watch.Reload.ConfigDestinationPath, err)
				return false
			}

			if len(watch.Reload.Cmd) > 0 {
				cmd := exec.Command(watch.Reload.Cmd[0], watch.Reload.Cmd[1:]...)
				output, err := cmd.CombinedOutput()
				if err != nil {
					glog.Warningln("Failed to reload:", watch.Reload.Cmd, err)
					return false
				}
				glog.Infoln("Output of config reload", string(output))
			}
		}

		return true // just keep watching TODO - add a way to control this behavior via input json
	})
}

func (this *Executor) Reload(watch *RegistryWatch) error {
	value_location, _, err := watch.Path, "", error(nil)
	if value_location == "" {
		value_location, _, err = RegistryKeyValue(KLiveWatch, watch)
		if err != nil {
			return err
		}
	}
	glog.Infoln("Getting value from", value_location)
	n, err := this.zk.Get(value_location)
	if err != nil {
		return err
	}
	containers_path, environments_path := ParseLiveValue(n.GetValueString())

	glog.Infoln("Reload config using values in", containers_path, "and environment in", environments_path)

	// fetch from containers path
	data, err := this.hostport_list_from_zk(containers_path, watch.MatchContainerPort)
	if err != nil {
		glog.Warningln("Failed to fetch containers from", containers_path, "Continue.")
		return err
	}

	configBuff, err := ExecuteTemplateUrl(this.zk, watch.Reload.ConfigUrl, this.AuthToken, data)
	if err != nil {
		return err
	}
	glog.V(100).Infoln("Config template:", string(configBuff))

	err = ioutil.WriteFile(watch.Reload.ConfigDestinationPath, configBuff, 0777)
	if err != nil {
		glog.Warningln("Cannot write config to", watch.Reload.ConfigDestinationPath, err)
		return err
	}

	if len(watch.Reload.Cmd) > 0 {
		cmd := exec.Command(watch.Reload.Cmd[0], watch.Reload.Cmd[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			glog.Warningln("Failed to reload:", watch.Reload.Cmd, err)
			return err
		}
		glog.Infoln("Output of config reload", string(output))
	}
	return nil
}

func (this *Executor) GetWatchAction(key string) (*RegistryWatch, error) {
	w, ok := this.watcher.GetRule(key).(*RegistryWatch)
	if !ok {
		return nil, errors.New("not-registry-watch")
	}
	return w, nil
}
