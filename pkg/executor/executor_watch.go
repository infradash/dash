package executor

import (
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/template"
	"github.com/qorio/maestro/pkg/zk"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// TODO - validation early -- before we get to here.

func pre_process(c *ConfigFile) {
	switch {
	case strings.Index(c.Path, "~/") == 0:
		c.Path = strings.Replace(c.Path, "~", os.Getenv("HOME"), 1)
	case strings.Index(c.Path, "./") == 0:
		c.Path = strings.Replace(c.Path, ".", os.Getenv("PWD"), 1)
	}
}

func (this *Executor) HandleConfigReload(cf *ConfigFile) error {
	pre_process(cf)
	glog.Infoln("Watching registry key", cf.Reload)

	if cf.Reload == nil {
		return nil
	}

	return this.watcher.AddWatcher(cf.Reload.Path(), cf, func(e zk.Event) bool {
		if e.State == zk.StateDisconnected {
			glog.Warningln(cf.Reload.Path(), "disconnected. No action.")
			return true // keep watching
		}
		this.Reload(cf)
		return true // just keep watching TODO - add a way to control this behavior via input json
	})
}

func (this *Executor) Reload(cf *ConfigFile) error {
	pre_process(cf)
	configBuff, err := template.ExecuteUrl(this.zk, cf.Url, this.AuthToken, this)
	if err != nil {
		glog.Infoln("Error:", err)
		return err
	}
	glog.V(100).Infoln("Config template:", string(configBuff))

	if len(cf.Path) > 0 {
		path := cf.Path
		if strings.Index(cf.Path, "file://") == 0 {
			path = cf.Path[len("file://"):]
		}
		err = ioutil.WriteFile(path, configBuff, 0777)
		if err != nil {
			glog.Warningln("Cannot write config to", cf.Path, err)
			return err
		}
		glog.Infoln("Written config to", cf.Path)
	}

	if len(cf.ReloadCmd) > 0 {
		cmd := exec.Command(cf.ReloadCmd[0], cf.ReloadCmd[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			glog.Warningln("Failed to reload:", cf.ReloadCmd, err)
			return err
		}
		glog.Infoln("Output of config reload", string(output))
	}
	return nil
}
