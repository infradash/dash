package dash

import (
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/zk"
	"sync"
)

type ZkWatcher struct {
	zk      zk.ZK `json:"-"`
	lock    sync.Mutex
	watches map[string]chan<- bool
	rules   map[string]interface{}
}

func NewZkWatcher(zk zk.ZK) *ZkWatcher {
	return &ZkWatcher{
		zk:      zk,
		watches: make(map[string]chan<- bool),
		rules:   make(map[string]interface{}),
	}
}

func (this *ZkWatcher) StopWatch(key string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if stop, has := this.watches[key]; has {
		glog.Infoln("Stopping current watch at", key)
		stop <- true
	}
}

func (this *ZkWatcher) AddWatcher(key string, rule interface{}, watcher func(e zk.Event) bool) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if stop, has := this.watches[key]; has {
		glog.Infoln("Stopping current watch at", key)
		stop <- true
	}
	glog.Infoln("Start watching registry at", key)
	stop, err := this.zk.KeepWatch(key, watcher)
	if err != nil {
		return err
	}

	this.watches[key] = stop
	this.rules[key] = rule
	return nil
}

func (this *ZkWatcher) GetRule(key string) interface{} {
	return this.rules[key]
}

func (this *ZkWatcher) UpdateRule(key string, rule interface{}) {
	this.rules[key] = rule
}
