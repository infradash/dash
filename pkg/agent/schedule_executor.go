package agent

import (
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/docker"
	"github.com/qorio/maestro/pkg/zk"
)

type ScheduleExecutor struct {
	Inbox chan<- []Job
	Stop  chan<- bool

	inbox chan []Job
	stop  chan bool

	zk     zk.ZK
	docker *docker.Docker
}

func NewScheduleExecutor(zk zk.ZK, docker *docker.Docker) *ScheduleExecutor {
	inbox, stop := make(chan []Job), make(chan bool)
	return &ScheduleExecutor{
		Inbox:  inbox,
		Stop:   stop,
		inbox:  inbox,
		stop:   stop,
		zk:     zk,
		docker: docker,
	}
}

func (this *ScheduleExecutor) Run() error {
	go func() {
		for {
			select {
			case actions := <-this.inbox:
				for i, action := range actions {
					glog.Infoln(i, "**************************************************")
					action.Execute(this.zk, this.docker)
				}

			case stop := <-this.stop:
				if stop {
					glog.Infoln("Stopping schedule executor")
					break
				}
			}
		}
	}()
	return nil
}
