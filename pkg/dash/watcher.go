package dash

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/zk"
	"sync"
)

var (
	alert_chan      = make(chan Alert)
	alert_chan_stop = make(chan bool)
)

func init() {
	if alert_chan == nil {
		panic("zk.Alert channel not ready")
	}

	go func() {
		for {
			select {
			case alert := <-alert_chan:
				if buff, err := json.Marshal(&alert); err == nil {
					glog.Infoln("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! ", string(buff))
				}
			case stop := <-alert_chan_stop:
				if stop {
					break
				}
			}
		}
	}()
}

type Alert struct {
	Error   error               `json:"error,omitemtpy"`
	Message string              `json:"message,omitempty"`
	Context interface{}         `json:"context,omitempty"`
	Func    func(zk.Event) bool `json:"-"`
}

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

func ZKWatcherAlerts() <-chan Alert {
	return alert_chan
}

func ZkWatcherStop() {

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
	stop, err := this.zk.KeepWatch(key, watcher, func(e error) {
		alert_chan <- Alert{Context: key, Func: watcher, Error: e, Message: e.Error()}
	})
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
