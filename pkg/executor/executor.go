package executor

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/task"
	"github.com/qorio/maestro/pkg/zk"
	"github.com/qorio/omni/runtime"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

type Executor struct {
	Identity

	QualifyByTags
	ZkSettings
	EnvSource

	StartTimeUnix int64
	Name          string

	NoSourceEnv bool

	// Options for controlling stdout
	WriteStdout        bool // write to stdout
	EscapeWhiteSpaces  bool
	Newline            bool
	QuoteChar          string
	GenerateBashExport bool

	Host string   `json:"host"`
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`

	ExecCmd  string   `json:"exec_cmd"`
	ExecArgs []string `json:"exec_args"`

	Initializer *ConfigLoader `json:"config_loader"`

	Daemon                   bool   `json:"daemon"`
	IgnoreChildProcessFails  bool   `json:"ignore_child_process_fails"`
	CustomVarsCommaSeparated string `json:"custom_vars"` // K1=E1,K2=E2,...

	MQTTConnectionTimeout       time.Duration `json:"mqtt_connection_timeout"`
	MQTTConnectionRetryWaitTime time.Duration `json:"mqtt_connection_wait_time"`
	TailFileOpenRetries         int           `json:"tail_file_open_retries"`
	TailFileRetryWaitTime       time.Duration `json:"tail_file_retry_wait_time"`

	// From maestro's orchestration
	Task    *task.Task
	runtime *task.Runtime

	// e.g. [ 'BOOT_TIME', '{{.StartTimestamp}}']
	// where the value is a template to apply to the state of the Exector object.
	customVars map[string]*template.Template

	ListenPort int          `json:"listen_port"`
	endpoint   http.Handler `json:"-"`

	zk zk.ZK `json:"-"`

	watcher *ZkWatcher

	// For storing the stdout of the command.  This is the result of the process
	processOutputBuffer bytes.Buffer
}

func (this *Executor) Stdin() io.Reader {
	if this.runtime == nil || this.runtime.Stdin() == nil {
		glog.Infoln("Sourcing process stdin from os.Stdin")
		return os.Stdin
	}
	glog.Infoln("Teeing input to os.Stderr:", this.Task.Stdin)
	//return io.TeeReader(io.MultiReader(this.runtime.Stdin(), os.Stdin), this.runtime.PublishStdin())
	// TODO - allow stdin to come through a topic subscriber.
	return io.TeeReader(io.MultiReader(os.Stdin), this.runtime.Stderr())
}

func (this *Executor) Stdout() io.Writer {
	if this.runtime == nil || this.runtime.Stdout() == nil {
		glog.Infoln("Sending process stdout to os.Stdout.")
		return io.MultiWriter(os.Stdout, &this.processOutputBuffer)
	}
	return io.MultiWriter(os.Stdout, this.runtime.Stdout(), &this.processOutputBuffer)
}

func (this *Executor) Stderr() io.Writer {
	if this.runtime == nil || this.runtime.Stderr() == nil {
		glog.Infoln("Sending process stdout to os.Stderr")
		return os.Stderr
	}
	return io.MultiWriter(os.Stderr, this.runtime.Stderr())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (this *Executor) connect_zk() error {
	if this.zk != nil {
		return nil
	}
	zookeeper, err := zk.Connect(strings.Split(this.Hosts, ","), this.Timeout)
	if err != nil {
		return err
	}
	this.zk = zookeeper
	this.watcher = NewZkWatcher(this.zk)
	return nil
}

func (this *Executor) wait_for_process_finish(done chan error) {
	err := <-done
	glog.Infoln("Got done signal:", err, "runtime=", this.runtime)
	if this.runtime != nil {
		if err == nil {
			// write the entire stdout buffer to the output path
			zerr := this.runtime.Success(this.processOutputBuffer.String())
			glog.Infoln("Written to success Err=", zerr)
		} else {
			zerr := this.runtime.Error(err.Error())
			glog.Infoln("Written to error Err=", zerr)
		}
	}
	if err != nil && !this.IgnoreChildProcessFails {
		panic(err)
	}
}

func (this *Executor) Exec() error {

	this.StartTimeUnix = time.Now().Unix()
	this.Host, _ = os.Hostname()

	if err := this.ParseCustomVars(); err != nil {
		panic(err)
	}

	var source func() ([]string, map[string]string) = nil
	vars := make([]string, 0)
	env := make(map[string]string)

	if this.NoSourceEnv {
		glog.Infoln("Not sourcing environment variables.")
	} else {
		glog.Infoln("Sourcing environment variables.")
		if this.ReadStdin {
			source = this.EnvFromStdin()
		} else {
			must(this.connect_zk())
			source = this.EnvFromZk()
		}
		vars, env = source()
	}

	// Inject additional environments
	vars, err := this.injectCustomVars(env)
	if err != nil {
		panic(err)
	}

	for _, k := range vars {
		value := env[k]
		os.Setenv(k, value)

		if this.EscapeWhiteSpaces && strings.ContainsAny(value, " \t\n") {
			value = strings.Replace(value, " ", "\\ ", -1)
		}

		format, delim, prefix := "%s%s=%s%s%s%s", " ", ""

		if this.Newline {
			delim = "\n"
		}
		if this.GenerateBashExport {
			prefix = "export "
		}
		if this.WriteStdout {
			fmt.Fprintf(os.Stdout, format, prefix, k, this.QuoteChar, value, this.QuoteChar, delim)
		}
	}

	if this.Initializer != nil {
		glog.Infoln("Loading configuration from", this.Initializer.ConfigUrl)
		// set up the context for applying the config as a template
		this.Initializer.Context = this

		executorConfig := new(ExecutorConfig)
		loaded, err := this.Initializer.Load(executorConfig, this.AuthToken, this.zk)
		if err != nil {
			panic(err)
		}
		if loaded {
			this.Task = executorConfig.Task

			if len(executorConfig.RegistryWatch) > 0 {
				must(this.connect_zk())
			}
			for _, w := range executorConfig.RegistryWatch {
				glog.Infoln("Configuring watch", w)
				err := this.SaveWatchAction(&w)
				if err != nil {
					panic(err)
				}
			}
			for _, t := range executorConfig.TailRequest {
				this.HandleTailRequest(&t)
			}
		}
	}

	if this.Task != nil {
		if this.Task.Id == "" {
			this.Task.Id = this.Id
		}
		glog.Infoln("Starting Task", "Id=", this.Task.Id)
		this.runtime, err = this.Task.Init(this.zk)
		if err != nil {
			panic(err)
		}
		_, _, err = this.runtime.Start()
		if err != nil {
			panic(err)
		}
	}

	if this.Cmd != "" {

		glog.Infoln("Processing with environment:", this.Cmd, this.Args)

		// Perform variable substitutions on the command and args
		if err := this.applyCmdSubstitutions(env); err != nil {
			panic(err)
		}

		cmd := exec.Command(this.ExecCmd, this.ExecArgs...)
		glog.Infoln("Starting", cmd.Path, cmd.Args, "in", cmd.Dir)

		// Wiring the input stream -- this will allows interactive console like bash
		cmd.Stdin = this.Stdin()
		cmd.Stdout = this.Stdout()
		cmd.Stderr = this.Stderr()

		// Run this in a closure and send a signal when done.
		process_done := make(chan error)

		go func() {
			cmd.Start()

			// Wait for cmd to complete even if we have no more stdout/stderr
			if cmd.Wait(); err != nil {
				process_done <- err
				return
			}

			ps := cmd.ProcessState
			if ps == nil {
				process_done <- errors.New(fmt.Sprintf("NoSuchCmd: %s %s", cmd.Path, cmd.Args))
				return
			}

			glog.Infoln("Process pid=", ps.Pid(), "Exited=", ps.Exited(), "Success=", ps.Success())

			if !ps.Success() {
				process_done <- errors.New(fmt.Sprintf("ProcessFailed: %s %s", cmd.Path, cmd.Args))
				return
			} else {
				process_done <- nil
				return
			}
		}()
		if !this.Daemon {
			this.wait_for_process_finish(process_done)
			return nil
		} else {

			// Run the process wait separately.. so we can get the signal
			go func() {
				this.wait_for_process_finish(process_done)
			}()

			// Keep this waiting... since the subprocess may have forked
			endpoint, err := NewApiEndPoint(this)
			if err != nil {
				panic(err)
			}

			this.endpoint = endpoint
			// This will block
			runtime.MinimalContainer(this.ListenPort,
				func() http.Handler {
					return endpoint
				},
				func() error {
					err := endpoint.Stop()
					glog.Infoln("Stopped endpoint", err)

					if this.zk != nil {
						err = this.zk.Close()
						glog.Infoln("Stopped zk", err)
					}
					return err
				})

		}
	}

	if this.runtime != nil {
		glog.Infoln("Finishing: stopping runtime")
		this.runtime.Stop()
	}
	return nil
}
