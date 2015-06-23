package registry

import (
	"flag"
	"time"
)

func (this *Registry) BindFlags() {
	flag.BoolVar(&this.Release, "release", false, "True to publish release record")
	flag.BoolVar(&this.Setlive, "setlive", false, "True to update record")
	flag.BoolVar(&this.SetliveNoWait, "setlive_nowait", false, "True to not wait")
	flag.BoolVar(&this.Commit, "commit", false, "True to commit the record")
	flag.BoolVar(&this.ReadValue, "read", false, "True to read value from registry")
	flag.StringVar(&this.ReadValuePath, "readpath", "", "The path to read value from")
	flag.IntVar(&this.SetliveMinThreshold, "setlive_min_instances", 1, "Minimal available instances before setlive.")
	flag.DurationVar(&this.SetliveWait, "setlive_wait", time.Duration(1*time.Minute), "Wait internval to check available instances.")
	flag.DurationVar(&this.SetliveMaxWait, "setlive_maxwait", time.Duration(5*time.Minute), "Setlive: max wait before giving up.")

	flag.StringVar(&this.WriteValue, "writevalue", "", "The value to write")
	flag.StringVar(&this.WriteValuePath, "writepath", "", "The path to write")

	flag.IntVar(&this.Retries, "retries", 5, "Retries")
	flag.IntVar(&this.RetriesWaitSeconds, "retries_wait_seconds", 5, "Wait seconds between retries")

	flag.StringVar(&this.SchedulerTriggerPath, "scheduler_trigger_path", "", "Scheduler trigger path; value is a counter")
	flag.StringVar(&this.SchedulerImagePath, "scheduler_image_path", "", "Scheduler image path; value is the image.")
}
