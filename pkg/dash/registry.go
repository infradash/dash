package dash

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/zk"
	"strings"
	"time"
)

type Registry struct {
	ZkSettings
	RegistryReleaseEntry

	// For arbitrary read access
	ReadValue     bool
	ReadValuePath string

	WriteValue     string
	WriteValuePath string

	// For release code version
	Release bool

	// For setting live version
	Setlive             bool
	SetliveNoWait       bool
	SetliveMinThreshold int           `json:"setlive_min_instances"`
	SetliveWait         time.Duration `json:"setlive_wait"`
	SetliveMaxWait      time.Duration `json:"setlive_max_wait"`

	Commit bool

	zk zk.ZK
}

func (this *Registry) check_input() error {
	if this.ReadValue && this.ReadValuePath != "" {
		return nil
	}

	if this.WriteValue != "" && this.WriteValuePath != "" {
		return nil
	}

	switch {
	case this.Domain == "":
		return errors.New("no-domain")
	case this.Service == "":
		return errors.New("no-service")
	case this.Version == "":
		return errors.New("no-version")
	}
	return nil
}

func (this *Registry) Connect() {
	zookeeper, err := zk.Connect(strings.Split(this.Hosts, ","), this.Timeout)
	if err != nil {
		panic(err)
	}
	this.zk = zookeeper
}

func (this *Registry) PrintReadPathValue() error {
	if this.zk == nil {
		this.Connect()
	}
	n, err := this.zk.Get(this.ReadValuePath)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(n.Value))
	return nil
}

func (this *Registry) GetReadPathValue() string {
	if this.zk == nil {
		this.Connect()
	}
	n, err := this.zk.Get(this.ReadValuePath)
	if err != nil {
		panic(err)
	}
	return (string(n.Value))
}

func (this *Registry) Run() error {
	if this.zk == nil {
		this.Connect()
	}

	if err := this.check_input(); err != nil {
		panic(err)
	}

	if this.ReadValue {
		this.PrintReadPathValue()
	}

	if this.WriteValuePath != "" && this.WriteValue != "" {
		glog.Infoln("Setting", this.WriteValuePath, "to", this.WriteValue)
		if this.Commit {
			err := create_or_set(this.zk, this.WriteValuePath, this.WriteValue)
			if err != nil {
				panic(err)
			} else {
				glog.Infoln("Committed", this.WriteValuePath)
			}
		}
		return nil
	}

	if this.Release {

		// The actual release entry
		key, value, err := RegistryKeyValue(KRelease, this)
		glog.Infoln("Releasing", key, "to", value, "err", err)
		if err != nil {
			return err
		}

		if this.Commit {
			err := create_or_set(this.zk, key, value)
			if err != nil {
				return err
			}
			glog.Infoln("Committed", key, "to", value)
		}

		// Now set the top level node
		key, value, err = RegistryKeyValue(KReleaseWatch, this)
		glog.Infoln("Release: Updating", key, "to", value, "err", err)

		if this.Commit {
			err := create_or_set(this.zk, key, value)
			if err != nil {
				return err
			}
			glog.Infoln("Release: Committed", key, "to", value)
		} else {
			glog.Infoln("Release: Skipped setting", key, "to", value)
		}
	}

	if this.Setlive {

		liveEntry := RegistryLiveEntry{
			RegistryReleaseEntry: this.RegistryReleaseEntry,
			Live:                 this.Commit,
		}
		key, value, err := RegistryKeyValue(KLive, liveEntry)
		glog.Infoln("Setting live:", key, "to", value, "err", err)
		if err != nil {
			return err
		}
		watch_key, _, err := RegistryKeyValue(KLiveWatch, liveEntry)
		glog.Infoln("Setting live - watch node:", watch_key, "err", err)
		if err != nil {
			return err
		}

		if liveEntry.Live {

			if !this.SetliveNoWait {
				var waited time.Duration
				for poll := true; poll; {
					// Check what's pointed at (the value) and make sure there are enough
					// instances to meet the threshold
					containers_path, _ := ParseLiveValue(value)
					glog.Infoln("Checking for instances under", containers_path)
					cp, err := this.zk.Get(containers_path)

					switch {
					case err == zk.ErrNotExist:
						glog.Infoln("Node", value, "not found. Check later.")

						time.Sleep(this.SetliveWait)
						waited += this.SetliveWait
						if waited >= this.SetliveMaxWait {
							return errors.New(fmt.Sprintf("Setlive: Timeout waiting for instances in %s", containers_path))
						}
					case int(cp.Stats.NumChildren) < this.SetliveMinThreshold:
						glog.Infoln("Instances count", cp.Stats.NumChildren,
							"less than threshold", this.SetliveMinThreshold, "Check later")

						time.Sleep(this.SetliveWait)
						waited += this.SetliveWait
						if waited >= this.SetliveMaxWait {
							return errors.New(fmt.Sprintf("Setlive: Timeout waiting for instances in %s", containers_path))
						}

					default:
						glog.Infoln("Found", cp.Stats.NumChildren, "instances. Continue")
						poll = false
					}
				}
			}

			if this.Commit {
				err := create_or_set(this.zk, key, value)
				if err != nil {
					return err
				}
				glog.Infoln("Setlive: Committed", key, "to", value)

				// Now also update the watch node
				if err := increment(this.zk, watch_key, 1); err != nil {
					return err
				} else {
					glog.Infoln("Setlive: Updated watch node", watch_key)
				}
			} else {
				glog.Infoln("Setlive: Skipped setting", key, "to", value)
			}
		}
	}

	if this.zk != nil {
		this.zk.Close()
	}
	return nil
}
