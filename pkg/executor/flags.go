package executor

import (
	"flag"
	"time"
)

func (this *Executor) BindFlags() {
	flag.BoolVar(&this.NoSourceEnv, "no_source_env", false, "True to skip sourcing env")
	flag.BoolVar(&this.WriteStdout, "stdout", false, "Wrtie to stdout")
	flag.BoolVar(&this.EscapeWhiteSpaces, "escape", false, "Escape space")
	flag.BoolVar(&this.Newline, "newline", false, "New line")
	flag.BoolVar(&this.GenerateBashExport, "bash_export", false, "Bash export")
	flag.BoolVar(&this.Daemon, "daemon", false, "True to block as daemon: useful for docker.")
	flag.BoolVar(&this.IgnoreChildProcessFails, "ignore_child_process_fails", false, "True to ignore child process fail")

	flag.StringVar(&this.QuoteChar, "quote", "", "Quote character")
	flag.StringVar(&this.CustomVarsCommaSeparated, "custom_vars", "BOOT_TIMESTAMP={{.StartTimeUnix}}", "Custom variables")

	flag.IntVar(&this.ListenPort, "listen", 25658, "Listening port for executor")

	flag.DurationVar(&this.MQTTConnectionTimeout, "mqtt_connect_timeout", time.Duration(10*time.Minute), "MQTT connection timeout")
	flag.DurationVar(&this.MQTTConnectionRetryWaitTime, "mqtt_connect_retry_wait_time", time.Duration(1*time.Minute), "MQTT connection wait time before retry")
	flag.IntVar(&this.TailFileOpenRetries, "tail_file_open_retries", 0, "Tail file open retries")
	flag.DurationVar(&this.TailFileRetryWaitTime, "tail_file_open_retry_wait", time.Duration(2*time.Second), "Tail file open wait time before retry")
}
