package env

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/zk"
	"os"
	"strings"
)

type Env struct {
	ZkSettings
	EnvSource         // specifies input
	RegistryEntryBase // specifies output

	ReadStdin bool `json:"read_stdin"`

	Publish   bool
	Overwrite bool

	zk zk.ZK
}

func (this *Env) Run() error {

	var source func() ([]string, map[string]interface{}) = nil

	if this.ReadStdin {
		source = this.EnvFromReader(os.Stdin)
	} else {

		// Simple check to see if path and destination env path are the same
		check, _, _ := RegistryKeyValue(KEnvRoot, this)
		if this.Path == check {
			glog.Infoln("Source", this.Path, "and destination", check, "are the same. Nothing to do.")
			return nil
		}

		zookeeper, err := zk.Connect(strings.Split(this.Hosts, ","), this.Timeout)
		if err != nil {
			panic(err)
		}
		this.zk = zookeeper

		source = this.EnvFromZk(this.zk)
	}

	if !this.RegistryEntryBase.CheckRequires() {
		return errors.New("no-path")
	}

	// Now run it
	vars, env := source()

	if this.Publish && this.zk == nil {
		zookeeper, err := zk.Connect(strings.Split(this.Hosts, ","), this.Timeout)
		if err != nil {
			panic(err)
		}
		this.zk = zookeeper
	}

	for _, k := range vars {

		entry := &RegistryEnvEntry{RegistryEntryBase: this.RegistryEntryBase, EnvName: k, EnvValue: fmt.Sprintf("%s", env[k])}
		key, value, err := RegistryKeyValue(KEnv, entry)

		if err != nil {
			return err
		}

		if this.Publish {
			// Upsert
			n, err := this.zk.Get(key)
			switch {

			case err == zk.ErrNotExist:
				n, err = this.zk.Create(key, []byte(value))
				glog.Infoln("Created", key, "err=", err)

			case err != nil:
				glog.Warningln("Error upsert", key, "err=", err)
				continue

			}

			if value != n.GetValueString() {

				if this.Overwrite {
					if err := n.Set([]byte(value)); err != nil {
						glog.Warningln("Error upsert", key, "err=", err)
					} else {
						glog.Warningln("Committed", key, "err=", err)
					}
				} else {
					glog.Infoln("No overwrite -- key=", key, "source=", value, "to=", n.GetValueString())
				}

			} else {
				glog.Infoln("Key", key, "have same values. Nothing updated.")
			}

		} else {
			fmt.Fprintf(os.Stdout, "%s/%s=>%s\n", this.Path, k, key)
		}
	}

	if this.zk != nil {
		this.zk.Close()
	}
	return nil
}
