package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/task"
	mtemplate "github.com/qorio/maestro/pkg/template"
	"github.com/qorio/maestro/pkg/zk"
	"github.com/qorio/omni/common"
	"github.com/qorio/omni/runtime"
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

	Context string `json:"context,omitempty"`

	StartTimeUnix int64

	NoSourceEnv bool

	// e.g. [ 'BOOT_TIME', '{{.StartTimestamp}}']
	// where the value is a template to apply to the state of the Exector object.
	customVars map[string]*template.Template

	Host string   `json:"host"`
	Dir  string   `json:"dir"`
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`

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

	// Tail files
	MQTTConnectionTimeout       time.Duration `json:"mqtt_connection_timeout"`
	MQTTConnectionRetryWaitTime time.Duration `json:"mqtt_connection_wait_time"`
	TailFileOpenRetries         int           `json:"tail_file_open_retries"`
	TailFileRetryWaitTime       time.Duration `json:"tail_file_retry_wait_time"`
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

func (this *Executor) load_context() (map[string]interface{}, error) {
	if this.Context == "" {
		return nil, nil
	}

	if strings.Index(this.Context, "env://") == 0 {
		err := this.connect_zk()
		if err != nil {
			return nil, err
		}
	}

	body, _, err := mtemplate.FetchUrl(this.Context, nil, this.zk)
	if err != nil {
		return nil, err
	}

	context := map[string]interface{}{}
	err = json.Unmarshal([]byte(body), &context)
	if err != nil {
		return nil, err
	}
	return context, nil
}

func (this *Executor) Exec() error {

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

	// Get any task context from a datasource url
	taskContext, err := this.load_context()
	if err != nil {
		panic(err)
	}

	var taskFromInitializer *task.Task

	if this.Initializer != nil {
		glog.Infoln("Loading configuration from", this.Initializer.ConfigUrl)
		// set up the context for applying the config as a template
		c := map[string]interface{}{
			"Task": this,
			"Env":  env,
		}
		if taskContext != nil {
			c["Context"] = taskContext
		}

		this.Initializer.Context = c

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
				this.HandleConfigReload(&c)
			}
			for _, t := range executorConfig.TailFiles {
				this.HandleTailFile(&t)
			}
		}
	}

	// Default task based on what's entered in the command line, which takes precedence.
	target := task.Task{
		Id: this.Id,
		Cmd: &task.Cmd{
			Dir:  this.Dir,
			Path: this.Cmd,
			Args: this.Args,
			Env:  envlist,
		},
		ExecOnly: this.ExecOnly,
	}

	if taskFromInitializer != nil {

		merged, err := taskFromInitializer.Copy()
		if err != nil {
			panic(err)
		}

		// What's specified in the command line wins
		merged.Id = target.Id

		if this.Cmd != "" {
			merged.Cmd = target.Cmd
		}

		target = *merged
	}

	// One final pass of applying taskContext to the command as if the command is a template:
	if taskContext != nil {
		applied := task.Cmd{}
		err := ApplyVarSubs(target.Cmd, &applied, map[string]interface{}{
			"Task":    this,
			"Env":     env,
			"Context": taskContext,
		})
		if err != nil {
			panic(err)
		}
		target.Cmd = &applied
	}

	var exit chan error

	if this.Daemon {
		exit = make(chan error)
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

					exit <- err
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

	if exit != nil {
		glog.Infoln("Waiting for API server to complete.")
		return <-exit
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

func (this *Executor) ParseCustomVars() error {
	this.customVars = make(map[string]*template.Template)

	for _, expression := range strings.Split(this.CustomVarsCommaSeparated, ",") {
		parts := strings.Split(expression, "=")
		if len(parts) != 2 {
			return ErrBadTemplate
		}
		key, exp := parts[0], parts[1]
		if t, err := template.New(key).Parse(exp); err != nil {
			return err
		} else {
			this.customVars[key] = t
		}
	}
	return nil
}

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
