package registry

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/registry"
	"github.com/qorio/maestro/pkg/zk"
	"strings"
	"time"
)

type Registry struct {
	ZkSettings
	RegistryReleaseEntry

	// Retries
	Retries            int `json:"retries,omitempty"`
	RetriesWaitSeconds int `json:"retries_wait_seconds,omitempty"`

	// V2 release syntax
	SchedulerTriggerPath string `json:"scheduler_trigger_path,omitempty"`
	SchedulerImagePath   string `json:"scheduler_image_path,omitempty"`

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

	if this.SchedulerImagePath == "" && this.SchedulerTriggerPath == "" {
		switch {
		case this.Domain == "":
			return errors.New("no-domain")
		case this.Service == "":
			return errors.New("no-service")
		case this.Version == "":
			return errors.New("no-version")
		}
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

func (this *Registry) retry_operation(f func() error) error {
	for i := 0; i < this.Retries+1; i++ {
		err := f()
		if err == nil {
			return nil
		} else {
			glog.Warningln("Operation failed. Err=", err, "Retrying. Attempt=", i)
			time.Sleep(time.Duration(this.RetriesWaitSeconds) * time.Second)
		}
	}
	glog.Infoln("Operation failed after retries")
	return errors.New("too-many-retries")
}

func (this *Registry) PrintReadPathValue() error {
	if this.zk == nil {
		this.Connect()
	}
	n, err := this.zk.Get(this.ReadValuePath)
	if err != nil {
		return err
	}
	fmt.Println(string(n.Value))
	return nil
}

func (this *Registry) Finish() {
	if this.zk != nil {
		glog.Infoln("Closing zk")
		this.zk.Close()
	}
}

func (this *Registry) Run() error {

	defer this.Finish()

	if this.zk == nil {
		this.Connect()
	}

	if err := this.check_input(); err != nil {
		panic(err)
	}

	if this.ReadValue {
		glog.Infoln("Reading", this.ReadValuePath)
		if err := this.retry_operation(func() error {
			return this.PrintReadPathValue()
		}); err != nil {
			panic(err)
		}
	}

	if this.WriteValuePath != "" && this.WriteValue != "" {
		if this.Commit {
			glog.Infoln("Setting", this.WriteValuePath, "to", this.WriteValue)
			if err := this.retry_operation(func() error {
				return zk.CreateOrSet(this.zk, registry.Path(this.WriteValuePath), this.WriteValue)
			}); err != nil {
				panic(err)
			}
		}
		return nil
	}

	if this.Release {

		// Support for v2 Scheduler triggers
		if this.SchedulerTriggerPath != "" && this.SchedulerImagePath != "" {
			if this.Commit {
				if err := this.retry_operation(func() error {
					glog.Infoln("Release updating scheduler image path:", this.SchedulerImagePath, "Image=", this.Image)
					if err1 := zk.CreateOrSet(this.zk, registry.Path(this.SchedulerImagePath), this.Image); err1 != nil {
						return err1
					}
					glog.Infoln("Release updating scheduler trigger path:", this.SchedulerTriggerPath)
					if err2 := zk.Increment(this.zk, registry.Path(this.SchedulerTriggerPath), 1); err2 != nil {
						return err2
					}
					return nil
				}); err != nil {
					return err
				}
			}
		}

		if this.Domain != "" && this.Service != "" && this.Version != "" {
			// Support for legacy notation
			if this.Commit {
				if err := this.retry_operation(func() error {
					// The actual release entry
					key, value, err := RegistryKeyValue(KRelease, this)
					glog.Infoln("Releasing", key, "to", value, "err", err)
					if err != nil {
						return err
					}
					if err1 := zk.CreateOrSet(this.zk, registry.Path(key), value); err1 != nil {
						return err1
					}
					// Now set the top level node
					key, value, err = RegistryKeyValue(KReleaseWatch, this)
					glog.Infoln("Release: Updating", key, "to", value, "err", err)
					if err != nil {
						return err
					}
					if err2 := zk.CreateOrSet(this.zk, registry.Path(key), value); err2 != nil {
						return err2
					}
					glog.Infoln("Committed", key, "to", value)
					return nil

				}); err != nil {
					return err
				}
			} else {
				glog.Infoln("Release: Skipped.")
			}
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
				if err := this.retry_operation(func() error {
					return zk.CreateOrSet(this.zk, registry.Path(key), value)
				}); err != nil {
					return err
				}

				glog.Infoln("Setlive: Committed", key, "to", value)

				// Now also update the watch node
				if err := this.retry_operation(func() error {
					return zk.Increment(this.zk, registry.Path(watch_key), 1)
				}); err != nil {
					return err
				} else {
					glog.Infoln("Setlive: Updated watch node", watch_key)
				}
			} else {
				glog.Infoln("Setlive: Skipped setting", key, "to", value)
			}
		}
	}

	return nil
}
