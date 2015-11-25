package executor

import (
	"flag"
	"time"
)

func (this *Executor) BindFlags() {
	flag.StringVar(&this.Context, "context", "", "Context of this task; a json resource location.")
	flag.IntVar(&this.Runs, "runs", 0, "Number of command executions; -1 means indefinite; > 0 is finite times.")
	flag.BoolVar(&this.NoSourceEnv, "no_source_env", false, "True to skip sourcing env")
	flag.BoolVar(&this.Daemon, "daemon", false, "True to start api server.")
	flag.BoolVar(&this.IgnoreChildProcessFails, "ignore_child_process_fails", false, "True to ignore child process fail")
	flag.StringVar(&this.CustomVarsCommaSeparated, "custom_vars", "BOOT_TIMESTAMP={{.StartTimeUnix}}", "Custom variables")
	flag.IntVar(&this.TimeoutSeconds, "timeout_seconds", -1, "Timeout in seconds")
	flag.IntVar(&this.ListenPort, "listen", 25658, "Listening port for executor")
	flag.StringVar(&this.Dir, "work_dir", "", "Working directory to execute the cmd")
	flag.BoolVar(&this.ExecOnly, "exec_only", false, "True for exec only -- no orchestration pubsub or znode semaphores")
	flag.DurationVar(&this.MQTTConnectionTimeout, "mqtt_connect_timeout", time.Duration(10*time.Minute), "MQTT connection timeout")
	flag.DurationVar(&this.MQTTConnectionRetryWaitTime, "mqtt_connect_retry_wait_time", time.Duration(1*time.Minute), "MQTT connection wait time before retry")
	flag.IntVar(&this.TailFileOpenRetries, "tail_file_open_retries", 0, "Tail file open retries")
	flag.DurationVar(&this.TailFileRetryWaitTime, "tail_file_open_retry_wait", time.Duration(2*time.Second), "Tail file open wait time before retry")
}
