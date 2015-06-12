package executor

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/mqtt"
	"github.com/qorio/maestro/pkg/pubsub"
	"github.com/qorio/maestro/pkg/registry"
	"github.com/qorio/maestro/pkg/workflow"
	"github.com/qorio/maestro/pkg/zk"
	"github.com/qorio/omni/runtime"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type Executor struct {
	QualifyByTags
	ZkSettings
	EnvSource

	StartTimeUnix int64

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
	Task *workflow.Task

	// e.g. [ 'BOOT_TIME', '{{.StartTimestamp}}']
	// where the value is a template to apply to the state of the Exector object.
	customVars map[string]*template.Template

	ListenPort int          `json:"listen_port"`
	endpoint   http.Handler `json:"-"`

	zk zk.ZK `json:"-"`

	watcher *ZkWatcher
}

func (this *Executor) EnvFromStdin() func() ([]string, map[string]string) {
	return func() ([]string, map[string]string) {
		keys := make([]string, 0)
		env := make(map[string]string)
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			i := strings.Index(line, "=")
			if i > 0 {
				key := line[0:i]
				value := line[i+1:]
				keys = append(keys, key)
				env[key] = value
			}
		}
		sort.Strings(keys)
		return keys, env
	}
}

func (this *Executor) EnvFromZk() func() ([]string, map[string]string) {
	return func() ([]string, map[string]string) {

		if !this.EnvSource.RegistryEntryBase.CheckRequires() {
			panic(errors.New("no-path"))
		}

		var env_path string
		if this.Path != "" {
			env_path = this.Path
		} else {
			key, _, err := RegistryKeyValue(KEnvRoot, this.EnvSource.RegistryEntryBase)
			if err != nil {
				panic(err)
			}
			env_path = key
		}

		glog.Infoln("Loading env from", env_path)

		root_node, err := this.zk.Get(env_path)
		if err != nil {
			panic(err)
		}

		// Just get the entire set of values and export them as environment variables
		all, err := root_node.FilterChildrenRecursive(func(z *zk.Node) bool {
			return !z.IsLeaf() // filter out parent nodes
		})

		if err != nil {
			panic(err)
		}

		keys := make([]string, 0)
		env := make(map[string]string)
		for _, node := range all {
			key, value, err := zk.Resolve(this.zk, registry.Path(node.GetBasename()), node.GetValueString())
			if err != nil {
				panic(errors.New("bad env reference:" + key.String() + "=>" + value))
			}

			env[key.String()] = value
			keys = append(keys, key.String())
		}
		sort.Strings(keys)
		return keys, env
	}
}

func (this *Executor) ParseCustomVars() error {
	this.customVars = make(map[string]*template.Template)

	for _, expression := range strings.Split(this.CustomVarsCommaSeparated, ",") {
		parts := strings.Split(expression, "=")
		if len(parts) != 2 {
			return errors.New("invalid-template:" + expression)
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

func (this *Executor) injectCustomVars(env map[string]string) ([]string, error) {
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

// Given the environments apply any substitutions to the command and args
// the format is same as template expressions e.g. {{.RUN_BINARY}}
// This allows the environment to be passed to the command even if the child process
// does not look at environment variables.
func (this *Executor) applyCmdSubstitutions(env map[string]string) error {
	templates := make([]*template.Template, len(this.Args)+1)
	templates[0] = template.Must(template.New(this.Cmd).Parse(this.Cmd))
	for i, a := range this.Args {
		templates[i+1] = template.Must(template.New(a).Parse(a))
	}

	// Now apply values
	applied := make([]string, len(templates))
	for i, t := range templates {
		var buff bytes.Buffer
		if err := t.Execute(&buff, env); err != nil {
			return err
		} else {
			applied[i] = buff.String()
		}
	}

	this.ExecCmd = applied[0]
	this.ExecArgs = applied[1:]
	glog.Infoln("ExecCmd=", this.ExecCmd, "ExecArgs=", this.ExecArgs)
	return nil
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

func (this *Executor) Exec() error {

	this.StartTimeUnix = time.Now().Unix()
	this.Host, _ = os.Hostname()

	if err := this.ParseCustomVars(); err != nil {
		panic(err)
	}

	var source func() ([]string, map[string]string) = nil
	vars := make([]string, 0)
	env := make(map[string]string)

	if !this.NoSourceEnv {
		glog.Infoln("Sourcing environment variables.")
		if this.ReadStdin {
			source = this.EnvFromStdin()
		} else {
			must(this.connect_zk())
			source = this.EnvFromZk()
		}
		vars, env = source()
	} else {
		glog.Infoln("Not sourcing environment variables.")
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

		format := "%s%s=%s%s%s%s"
		delim := " "
		prefix := ""
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
		glog.Infoln("Loading configuration from", this.Initializer.SourceUrl)
		// set up the context for applying the config as a template
		this.Initializer.Context = this

		executorConfig := new(ExecutorConfig)
		err := this.Initializer.Load(executorConfig)
		if err != nil {
			panic(err)
		}

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

		if executorConfig.Task != nil {
			this.Task = executorConfig.Task
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
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run this in a closure and send a signal when done.
		process_done := make(chan bool)

		go func() {
			defer func() { process_done <- true }()

			cmd.Start()

			// Wait for cmd to complete even if we have no more stdout/stderr
			if cmd.Wait(); err != nil {
				panic(fmt.Sprintf("Child process failed:%s", err))
			}

			ps := cmd.ProcessState
			if ps == nil {
				panic(fmt.Sprintf("No process for %s %s", cmd.Path, cmd.Args))
			}

			glog.Infoln("Process pid=", ps.Pid(), "Exited=", ps.Exited(), "Success=", ps.Success())

			if !this.IgnoreChildProcessFails && !ps.Success() {
				panic("Child process failed.")
			}
		}()
		if !this.Daemon {
			<-process_done // Wait till the process completes
			return nil
		} else {
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

	return nil
}

func (this *Executor) SaveWatchAction(watch *RegistryWatch) error {
	watch_key, _, err := watch.Path, "", error(nil)
	if watch_key == "" {
		watch_key, _, err = RegistryKeyValue(KLiveWatch, watch)
		if err != nil {
			return err
		}
	}

	glog.Infoln("Watching registry key", watch_key)

	return this.watcher.AddWatcher(watch_key, watch, func(e zk.Event) bool {

		glog.Infoln("Observed event for key", watch_key, e)

		if e.State == zk.StateDisconnected {
			glog.Warningln(watch_key, "disconnected. No action.")
			return true // keep watching
		}

		value_location := watch_key
		// read the value -- do we have an actual value node that we should read from?
		if watch.ValueLocation != nil {
			if watch.ValueLocation.Path != "" {
				value_location = watch.ValueLocation.Path
			} else {
				v, _, err := RegistryKeyValue(KLive, watch.ValueLocation)
				if err != nil {
					glog.Warningln("No value location. Stop watching", *watch)
					this.watcher.StopWatch(watch_key)
					return false
				}
				value_location = v
			}
		}

		glog.Infoln("Getting value from", value_location)
		n, err := this.zk.Get(value_location)
		if err != nil {
			glog.Warningln("Dropping watch of", watch_key, "on error", err)
			return false
		}

		glog.Infoln("Observed change at", watch_key, "location=", value_location, "value=", n.GetValueString())

		containers_path, environments_path := ParseLiveValue(n.GetValueString())

		if watch.ReloadConfig != nil {
			glog.Infoln("Reload config using values in", containers_path, "and environment in", environments_path)

			// fetch from containers path
			data, err := this.hostport_list_from_zk(containers_path, watch.MatchContainerPort)
			if err != nil {
				glog.Warningln("Failed to fetch containers from", containers_path, "Continue.")
				return true // Keep watching
			}

			configBuff, err := ExecuteTemplateUrl(this.zk, watch.ReloadConfig.ConfigTemplateUrl, data)
			if err != nil {
				return false
			}
			glog.V(100).Infoln("Config template:", string(configBuff))

			err = ioutil.WriteFile(watch.ReloadConfig.ConfigDestinationPath, configBuff, 0777)
			if err != nil {
				glog.Warningln("Cannot write config to", watch.ReloadConfig.ConfigDestinationPath, err)
				return false
			}

			if len(watch.ReloadConfig.ReloadCmd) > 0 {
				cmd := exec.Command(watch.ReloadConfig.ReloadCmd[0], watch.ReloadConfig.ReloadCmd[1:]...)
				output, err := cmd.CombinedOutput()
				if err != nil {
					glog.Warningln("Failed to reload:", watch.ReloadConfig.ReloadCmd, err)
					return false
				}
				glog.Infoln("Output of config reload", string(output))
			}
		}

		return true // just keep watching TODO - add a way to control this behavior via input json
	})
}

func (this *Executor) ReloadConfig(watch *RegistryWatch) error {
	value_location, _, err := watch.Path, "", error(nil)
	if value_location == "" {
		value_location, _, err = RegistryKeyValue(KLiveWatch, watch)
		if err != nil {
			return err
		}
	}
	glog.Infoln("Getting value from", value_location)
	n, err := this.zk.Get(value_location)
	if err != nil {
		return err
	}
	containers_path, environments_path := ParseLiveValue(n.GetValueString())

	glog.Infoln("Reload config using values in", containers_path, "and environment in", environments_path)

	// fetch from containers path
	data, err := this.hostport_list_from_zk(containers_path, watch.MatchContainerPort)
	if err != nil {
		glog.Warningln("Failed to fetch containers from", containers_path, "Continue.")
		return err
	}

	configBuff, err := ExecuteTemplateUrl(this.zk, watch.ReloadConfig.ConfigTemplateUrl, data)
	if err != nil {
		return err
	}
	glog.V(100).Infoln("Config template:", string(configBuff))

	err = ioutil.WriteFile(watch.ReloadConfig.ConfigDestinationPath, configBuff, 0777)
	if err != nil {
		glog.Warningln("Cannot write config to", watch.ReloadConfig.ConfigDestinationPath, err)
		return err
	}

	if len(watch.ReloadConfig.ReloadCmd) > 0 {
		cmd := exec.Command(watch.ReloadConfig.ReloadCmd[0], watch.ReloadConfig.ReloadCmd[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			glog.Warningln("Failed to reload:", watch.ReloadConfig.ReloadCmd, err)
			return err
		}
		glog.Infoln("Output of config reload", string(output))
	}
	return nil
}

func (this *Executor) GetWatchAction(key string) (*RegistryWatch, error) {
	w, ok := this.watcher.GetRule(key).(*RegistryWatch)
	if !ok {
		return nil, errors.New("not-registry-watch")
	}
	return w, nil
}

// This executes asynchronously
func (this *Executor) HandleTailRequest(req *TailRequest) {
	tail := *req // copy
	go func() {
		var out io.Writer = os.Stdout
		switch {
		case tail.Output == "stderr":
			glog.Infoln("Tailing", tail.Path, "sending to stderr")
			out = os.Stderr
		case tail.Output == "mqtt":
			id := fmt.Sprintf("/%s/%s/%s", this.Domain, filepath.Base(tail.Path), this.Host)
			topic := id
			if tail.MQTTTopic != "" {
				topic = tail.MQTTTopic
			}

			glog.Infoln("Tailing", tail.Path, "sending to mqtt as", topic)
			var wait time.Duration
			var addr *string
			for {
				addr = zk.GetValue(this.zk, registry.Path(tail.RegistryPath))
				if addr != nil || wait >= this.MQTTConnectionTimeout {
					break
				} else {
					glog.Infoln("Waiting for MQTT broker to become available -- path=", tail.RegistryPath)
					time.Sleep(this.MQTTConnectionRetryWaitTime)
					wait += this.MQTTConnectionRetryWaitTime
				}
			}
			if addr == nil {
				glog.Warningln("Cannot locate MQTT broker. Give up. Path=", tail.RegistryPath)
				return
			}
			mqtt, err := mqtt.Connect(id, *addr)
			if err != nil {
				glog.Warningln("Error starting mqtt client. Err=", err,
					"Broker path=", tail.RegistryPath, "host:port=", *addr, "topic=", topic)
				return
			}
			out = pubsub.GetWriter(pubsub.Topic(topic), mqtt)
			glog.Infoln("MQTT client for", tail.Path, "topic=", topic, "ready", out)

		default:
			// other cases include a url for websocket connection to push the output
			glog.Infoln("Tailing", tail.Path, "sending to stdout")
		}
		this.TailFile(tail.Path, out)
	}()
}

func (this *Executor) TailFile(path string, outstream io.Writer) error {
	glog.Infoln("Tailing file", path, outstream)

	tailer := &Tailer{
		Path: path,
	}

	// results channel
	output := make(chan interface{})
	stop := make(chan bool)
	go func() {
		for {
			select {
			case line := <-output:
				fmt.Fprintf(outstream, "%s", line)
				glog.V(100).Infoln(path, "=>", fmt.Sprintf("%s", line))
			case term := <-stop:
				if term {
					return
				}
			}
		}
	}()

	// Start tailing
	stop_tail := make(chan bool)
	go func() {

		tries := 0
		for {
			err := tailer.Start(output, stop_tail)
			if err != nil {

				// This can go on indefinitely.  We want this behavior because some files
				// don't get written until requests come in.  So a file can be missing for a while.
				if this.TailFileOpenRetries > 0 && tries >= this.TailFileOpenRetries {
					glog.Warningln("Stopping trying to tail", path, "Attempts:", tries)
					break
				}
				glog.Warningln("Error while tailing", path, "Err:", err, "Attempts:", tries)
				time.Sleep(time.Duration(this.TailFileRetryWaitTime))
				tries++

			} else {
				glog.Infoln("Starting to tail", path)
				break // no error
			}
		}
		stop <- true // stop reading
	}()

	return nil
}

func (this *Executor) hostport_list_from_zk(containers_path string, service_port *int) (interface{}, error) {
	n, err := this.zk.Get(containers_path)
	if err != nil {
		return nil, err
	}
	all, err := n.VisitChildrenRecursive(func(z *zk.Node) bool {
		_, port := ParseHostPort(z.GetBasename())
		return z.IsLeaf() || (service_port != nil && port == strconv.Itoa(*service_port) && z.IsLeaf())
	})
	if err != nil {
		return nil, err
	}

	list := make([]interface{}, 0)
	for _, c := range all {
		host, port := ParseHostPort(c.GetValueString())
		list = append(list, struct {
			Host string
			Port string
		}{
			Host: host,
			Port: port,
		})
	}
	return struct {
		HostPortList []interface{}
	}{
		HostPortList: list,
	}, nil
}
