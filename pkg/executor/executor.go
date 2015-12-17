package executor

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/registry"
	"github.com/qorio/maestro/pkg/task"
	"github.com/qorio/maestro/pkg/zk"
	"github.com/qorio/omni/common"
	"github.com/qorio/omni/runtime"
	"github.com/qorio/omni/version"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"
)

type Executor struct {
	Identity

	QualifyByTags
	ZkSettings
	EnvSource

	task.Cmd

	Config *ExecutorConfig `json:"config,omitempty"`

	StartTimeUnix int64 `json:"start_time_unix"`

	NoSourceEnv bool `json:"no_source_env"`

	// e.g. [ 'BOOT_TIME', '{{.StartTimestamp}}']
	// where the value is a template to apply to the state of the Exector object.
	customVars map[string]*template.Template

	Host string `json:"host"`

	// Dir  string   `json:"dir"`
	// Cmd  string   `json:"cmd"`
	// Args []string `json:"args"`

	Initializer *ConfigLoader `json:"config_loader"`

	IgnoreChildProcessFails  bool   `json:"ignore_child_process_fails"`
	CustomVarsCommaSeparated string `json:"custom_vars"` // K1=E1,K2=E2,...

	Runs           int  `json:"runs"`
	Daemon         bool `json:"daemon"`
	TimeoutSeconds int  `json:"timeout_seconds"`
	ExecOnly       bool `json:"exec_only,omitempty"`

	ListenPort int          `json:"listen_port"`
	endpoint   http.Handler `json:"-"`

	zk zk.ZK `json:"-"`

	watcher *ZkWatcher

	exit chan error

	// Tail files
	MQTTConnectionTimeout       time.Duration `json:"mqtt_connection_timeout"`
	MQTTConnectionRetryWaitTime time.Duration `json:"mqtt_connection_wait_time"`
	TailFileOpenRetries         int           `json:"tail_file_open_retries"`
	TailFileRetryWaitTime       time.Duration `json:"tail_file_retry_wait_time"`
}

func (this *Executor) GetInfo() *Info {
	return &Info{
		Executor: this,
		Version:  *version.BuildInfo(),
		Environ:  os.Environ(),
	}
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

func (this *Executor) Stdin() io.Reader {
	return os.Stdin
}

// For sourcing additional environments specified in the config -- TODO - clean up this code
func (this *Executor) source_envs(envlist []string, env map[string]interface{}) []string {
	if this.Config == nil {
		return envlist
	}
	for _, s := range this.Config.Envs {
		es := &EnvSource{
			Url: s,
		}
		vars, kv := es.Source(this.AuthToken, this.zk)()
		for _, k := range vars {
			value := kv[k]
			os.Setenv(k, fmt.Sprintf("%s", value))
			envlist = append(envlist, fmt.Sprintf("%s=%s", k, kv[k]))
			env[k] = value
		}
	}
	return envlist
}

func (this *Executor) Exec() {

	if this.Id == "" {
		this.Id = common.NewUUID().String()
	}

	this.StartTimeUnix = time.Now().Unix()
	this.Host, _ = os.Hostname()

	if err := this.ParseCustomVars(); err != nil {
		panic(err)
	}

	vars := make([]string, 0)
	env := make(map[string]interface{})

	if this.NoSourceEnv || this.EnvSource.IsZero() {
		glog.Infoln("Not sourcing environment variables.  NoSourceEnv=", this.NoSourceEnv, "EnvSourceIsZero=", this.EnvSource.IsZero())
	} else {
		glog.Infoln("Sourcing environment variables.")
		must(this.connect_zk())
		vars, env = this.Source(this.AuthToken, this.zk)()
	}

	// Inject additional environments
	vars, err := this.InjectCustomVars(env)
	if err != nil {
		panic(err)
	}

	// Export the environment variables
	envlist := []string{}
	for _, k := range vars {
		value := env[k]
		os.Setenv(k, fmt.Sprintf("%s", value))
		envlist = append(envlist, fmt.Sprintf("%s=%s", k, env[k]))
	}

	var taskFromInitializer *task.Task
	if this.Initializer != nil {
		glog.Infoln("Loading configuration from", this.Initializer.ConfigUrl)
		this.Initializer.Context = map[string]interface{}{
			"name":    this.Name,
			"id":      this.Id,
			"domain":  this.Domain,
			"service": this.Service,
			"version": this.Version,
			"environ": env,
			"now":     time.Now().Unix(),
			"runtime": EscapeVars(
				"id",
				"name",
				"start",
				"exit",
				"status"),
		}
		executorConfig := new(ExecutorConfig)
		loaded, err := this.Initializer.Load(executorConfig, this.AuthToken, this.zk)
		if err != nil {
			panic(err)
		}

		if loaded {
			taskFromInitializer = &executorConfig.Task

			if len(executorConfig.ConfigFiles) > 0 {
				must(this.connect_zk())
			}
			for _, c := range executorConfig.ConfigFiles {

				// Set up any watch related to config reload
				this.HandleConfigReload(c)
			}

			// collect the tail files and topics
			tails := map[string]string{}
			for _, t := range executorConfig.TailFiles {
				this.HandleTailFile(t)

				if len(t.Topic) > 0 {
					tails[t.Path] = t.Topic.String()
				}
			}

			// register this
			if this.zk != nil {
				k := registry.NewPath(this.Domain, this.Service, "_logs", this.Host)
				err := zk.CreateOrSet(this.zk, k, tails, true)
				glog.Infoln("Registered tail topics:", k, err)
			}

			// apply any config files
			for _, c := range executorConfig.ConfigFiles {

				if c.Init {
					glog.Infoln("Initializing config. Url=", c.Url, "Description=", c.Description)
					// Initialize and load the config first.
					if err := this.Reload(c); err != nil {
						glog.Warningln("Error initializing config", c, "Err=", err)
						panic(err)
					}
				}
			}

			// mount filesystems
			if err := StartFileMounts(executorConfig.Mounts, this.zk); err != nil {
				panic(err)
			}

			this.Config = executorConfig
		}
	}

	envlist = this.source_envs(envlist, env)
	this.Cmd.Env = envlist

	// Default task based on what's entered in the command line, which takes precedence.
	target := task.Task{
		Id:       this.Id,
		Cmd:      &this.Cmd,
		ExecOnly: this.ExecOnly,
	}

	if taskFromInitializer != nil {
		merged, err := taskFromInitializer.Copy()
		if err != nil {
			panic(err)
		}
		merged.Id = target.Id
		if merged.Cmd != nil {
			glog.Infoln("Using cmd from config:", merged.Cmd)
		} else {
			merged.Cmd = &this.Cmd
			glog.Infoln("Using cmd from commadline:", merged.Cmd)
		}
		target = *merged
	}

	if this.Daemon {
		this.exit = make(chan error)
		go func() {
			glog.Infoln("Starting API server")
			endpoint, err := NewApiEndPoint(this)
			if err != nil {
				panic(err)
			}
			this.endpoint = endpoint
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

					glog.Infoln("Stopping file mounts")
					StopFileMounts()

					this.exit <- err
					return err
				})
		}()
	}

	runs := 1
	switch {
	case this.Runs != 0:
		runs = this.Runs
	case this.Runs == 0 && target.Runs != 0:
		runs = target.Runs
	}

	// Keep looping if
	for runs != 0 {

		glog.Infoln(runs, "Starting Task", "Id=", target.Id, "ExecOnly=", target.ExecOnly)
		if target.Cmd != nil {
			glog.Infoln("Cmd=", target.Cmd.Path, "Args=", target.Cmd.Args)
		}

		taskRuntime, err := target.Init(this.zk)
		if err != nil {
			panic(err)
		}

		taskRuntime.StdinInterceptor(func(in string) (out string, ok bool) {
			return in, strings.Index(in, "#quit") != 0
		})

		err = taskRuntime.ApplyEnvAndFuncs(env, nil)
		if err != nil {
			panic(err)
		}

		taskRuntime.CaptureStdout()

		if this.Config.Namespace != nil {
			glog.Infoln("Annoucing in namespace", this.Config.Namespace)
			taskRuntime.Announce() <- task.Announce{
				Key:       "running",
				Value:     target,
				Ephemeral: true,
			}

			taskRuntime.Announce() <- task.Announce{
				Key:       "info",
				Value:     this.GetInfo(),
				Ephemeral: false,
			}

		}

		done, err := taskRuntime.Start()
		if err != nil {
			glog.Fatalln("Cannot start", err)
		}

		// Set up timeout
		timer := time.NewTimer(0 * time.Second)
		timer.Stop()
		if this.TimeoutSeconds > 0 {
			timer.Reset(time.Duration(this.TimeoutSeconds) * time.Second)
		}

		this.exec_wait(done, timer.C)
		runs += -1

		if runs == 0 {
			glog.Infoln("Stopping runtime")
			taskRuntime.Stop()
		}
	}
}

func (this *Executor) Wait() error {
	if this.exit != nil {
		glog.Infoln("Daemon mode. Blocking wait.")
		return <-this.exit
	}
	return nil
}

func (this *Executor) exec_wait(done chan error, timeout <-chan time.Time) {
	select {
	case result := <-done:
		switch result {
		case task.ErrTimeout:
			panic(result)
		case nil:
			glog.Infoln("Success")
		default:
			if !this.IgnoreChildProcessFails {
				panic(result)
			}
		}
	case <-timeout:
		panic("timeout")
	}
}

// Command-line custom vars can be templates with ${var} for shell environment expansion.
// Parse these first.
func (this *Executor) ParseCustomVars() error {
	this.customVars = make(map[string]*template.Template)
	for _, expression := range strings.Split(this.CustomVarsCommaSeparated, ",") {
		mid := strings.Index(expression, "=")
		key := expression[0:mid]
		exp := os.ExpandEnv(expression[mid+1:]) // also expand based on environment variables

		glog.Infoln("Expanded from", expression[mid+1:], "to", exp)
		if t, err := template.New(key).Funcs(template.FuncMap{
			"env": func(k string) interface{} {
				return os.Getenv(k)
			},
		}).Parse(exp); err != nil {
			return err
		} else {
			this.customVars[key] = t
		}
	}
	return nil
}

// Evaluate all the custom var expressions based on the current state of the executor.
func (this *Executor) InjectCustomVars(env map[string]interface{}) ([]string, error) {
	for k, t := range this.customVars {
		var buff bytes.Buffer
		if err := t.Execute(&buff, this); err != nil {
			return nil, err
		} else {
			env[k] = buff.String()
			glog.Infoln("CustomVar:", k, buff.String())
		}
	}

	keys := make([]string, 0)
	for k, _ := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}
