package agent

import (
	"github.com/golang/glog"
	"math"
)

// global max, min, local max, min
func (this *SwarmSchedule) check() (int, int, int, int, error) {
	// Set boundaries
	globalMax := math.MaxInt64
	if this.MaxInstancesGlobal != nil {
		globalMax = *this.MaxInstancesGlobal
	}

	globalMin := 0
	if this.MinInstancesGlobal != nil {
		globalMin = *this.MinInstancesGlobal
	}

	localMax := globalMax
	if this.MaxInstancesPerHost != nil {
		localMax = *this.MaxInstancesPerHost
	}

	localMin := 0
	if this.MinInstancesPerHost != nil {
		localMin = *this.MinInstancesPerHost
	}

	// base error cases
	switch {
	case localMin > localMax:
		return 0, 0, 0, 0, ErrBadSchedulerConfig
	case localMin > globalMax:
		return 0, 0, 0, 0, ErrBadSchedulerConfig
	case localMax > globalMax:
		return 0, 0, 0, 0, ErrBadSchedulerConfig
	}

	return globalMax, globalMin, localMax, localMin, nil
}

func (this *SwarmSchedule) Schedule(localRunning, globalRunning int) (int, error) {
	count, err := this.schedule(localRunning, globalRunning)
	glog.Infoln("Current: Global=", globalRunning, "Local=", localRunning, "Scheduled=", count, "Err=", err)
	return count, err
}
func (this *SwarmSchedule) schedule(localRunning, globalRunning int) (int, error) {

	globalMax, globalMin, localMax, localMin, err := this.check()
	glog.Infoln("Limits: Global=[", globalMin, ",", globalMax, ")", "Local=[", localMin, ",", localMax, ")")
	if err != nil {
		return 0, err
	}

	if globalRunning >= globalMax {
		return 0, nil
	}

	if globalRunning < globalMax {
		// we should start... check local limit
		if localRunning < localMax {
			// ok to start...  start one
			return 1, nil
		}
	}

	return 0, nil
}
