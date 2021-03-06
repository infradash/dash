package agent

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
	_ "github.com/qorio/maestro/pkg/mqtt"
	"github.com/qorio/maestro/pkg/pubsub"
	"github.com/qorio/maestro/pkg/registry"
	"github.com/qorio/maestro/pkg/template"
	"github.com/qorio/maestro/pkg/zk"
	"github.com/qorio/omni/runtime"
	"github.com/qorio/omni/version"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type Agent struct {
	QualifyByTags

	ZkSettings
	DockerSettings

	RegistryContainerEntry

	ListenPort   int `json:"listen_port"`
	DockerUIPort int `json:"dockerui_port"`

	UiDocRoot string `json:"ui_doc_root,omitempty"`
	EnableUI  bool   `json:"enable_ui,omitempty"`

	Initializer *ConfigLoader `json:"config_loader"`

	selfRegister bool `json:"-"`

	// json skips these fields
	endpoint       http.Handler      `json:"-"`
	zk             zk.ZK             `json:"-"`
	docker         *docker.Docker    `json:"-"`
	self_container *docker.Container `json:"-"`

	lock       sync.Mutex
	configLock sync.Mutex

	domains       map[string]*Domain
	domainConfigs map[string]DomainConfig

	StatusPubsubTopic string `json:"status_topic,omitempty"`
	statusTopic       pubsub.Topic
}

// Checks that all the information required for agent start up is met.
func (this *Agent) checkPreconditions() {
	if this.Host == "" {
		panic(ErrNoHost)
	}
	if this.Name == "" {
		panic(ErrNoName)
	}
}

func (this *Agent) is_running_dockerui() bool {
	return this.EnableUI && this.UiDocRoot != ""
}

func (this *Agent) clear_state() error {
	defer this.lock.Unlock()
	this.lock.Lock()
	this.domains = make(map[string]*Domain)
	this.domainConfigs = make(map[string]DomainConfig)
	return nil
}

func (this *Agent) LoadConfig(config *ConfigLoader) error {
	defer this.configLock.Unlock()
	this.configLock.Lock()

	if config == nil {
		return nil
	}

	glog.Infoln("Loading configuration from", config.ConfigUrl)

	var list []DomainConfig
	_, err := config.Load(&list, this.AuthToken, this.zk)
	if err != nil {
		return err
	}

	glog.Infoln("Loaded and applied configuration. Processing.")

	this.clear_state()

	for _, per_domain := range list {
		applied := new(DomainConfig)
		err := ApplyVarSubs(per_domain, applied,
			MergeMaps(map[string]interface{}{
				"Domain": per_domain.Domain,
			}, EscapeVars(ConfigVariables[1:]...)))
		if err != nil {
			return err
		}
		per_domain = *applied

		if _, config_err := this.ConfigureDomain(&per_domain); config_err != nil {
			return err
		}
	}

	glog.Infoln("Start running discovery / container monitors")
	matcher := new(DiscoveryContainerMatcher).Init()
	for _, domain := range this.domains {
		watches, err := domain.GetContainerWatcherSpecs()
		if err != nil {
			return err
		}
		for svc, watch := range watches {

			glog.Infoln(domain.Domain, svc, "Container matcher for discovery:", *watch)
			matcher.C(domain.Domain, svc, watch)

			glog.Infoln(domain.Domain, svc, "Set up container monitor:", *watch)
			domain.WatchContainer(svc, watch)
		}
	}

	err = this.DiscoverRunningContainers(matcher.Match, this.onMatchContainer)
	if err != nil {
		glog.Infoln("Error discovering containers:", err)
		return err
	}

	// Configure and start up services
	glog.Infoln("Configure domains")
	for _, domain := range this.domains {
		glog.Infoln("Starting services: Domain=", domain.Domain)
		_, config_err := domain.StartServices(this.QualifyByTags)
		if config_err != nil {
			return config_err
		}
	}

	glog.Infoln("Synchronize local states with scheduler")
	for _, domain := range this.domains {
		err := domain.SynchronizeSchedule()
		if err != nil {
			glog.Warningln("Failed to synchronize scheduling for Domain=", domain.Identity, "Err=", err)
			return err
		}
	}

	return nil
}

func (this *Agent) StartDockerUI() <-chan error {
	serverError := make(chan error)
	mux := http.NewServeMux()
	glog.Infoln("Starting UI with docroot=", this.UiDocRoot, "DockerPort=", this.DockerPort)
	fileHandler := http.FileServer(http.Dir(this.UiDocRoot))

	dockerApiHandler := this.createDockerApiHandler(this.DockerPort)
	mux.Handle("/dockerapi/", http.StripPrefix("/dockerapi", dockerApiHandler))
	mux.Handle("/", fileHandler)
	go func() {
		p := fmt.Sprintf(":%d", this.DockerUIPort)
		glog.Infoln("UI Listening on", p)
		if err := http.ListenAndServe(p, mux); err != nil {
			glog.Warningln("Error starting DockerUI", err)
			serverError <- err
			return
		}
	}()
	return serverError
}

// Block until SIGTERM
func (this *Agent) Run() {

	this.checkPreconditions()

	glog.Infoln("Agent", this.GetIdentity())

	this.domains = make(map[string]*Domain, 0)
	this.domainConfigs = make(map[string]DomainConfig, 0)

	err := this.ConnectServices()
	if err != nil {
		panic(err)
	}

	endpoint, err := NewApiEndPoint(this)
	if err != nil {
		panic(err)
	}

	if this.is_running_dockerui() {
		serverError := this.StartDockerUI()
		go func() {
			err := <-serverError
			glog.Warningln("DockerUI server error", err)
		}()
	}

	if this.selfRegister {
		err = this.DiscoverSelfInDocker()
		if err != nil {
			panic(err)
		}

		err = this.Register()
		if err != nil {
			panic(err)
		}
	}

	this.endpoint = endpoint

	err = this.LoadConfig(this.Initializer)
	if err != nil {
		panic(err)
	}

	runtime.MinimalContainer(this.ListenPort,
		func() http.Handler {
			return endpoint
		},
		func() error {
			err := endpoint.Stop()
			glog.Infoln("Stopped endpoint", err)
			err = this.zk.Close()
			glog.Infoln("Stopped zk", err)
			return err
		})
}

func (this *Agent) GetIdentity() string {
	return fmt.Sprintf("%s:%s", this.Name, this.Host)
}

func (this *Agent) ConnectServices() error {
	glog.Infoln("Connecting to zookeeper:", this.Hosts)
	zc, err := zk.Connect(strings.Split(this.Hosts, ","), this.Timeout)
	if err != nil {
		return err
	}
	this.zk = zc
	glog.Infoln("Connected to zookeeper:", this.Hosts)

	glog.Infoln("Connecting to docker:", this.DockerSettings)
	if this.Cert != "" {

		docker, err := docker.NewTLSClient(this.DockerPort, this.Cert, this.Key, this.Ca)
		if err != nil {
			return err
		}
		this.docker = docker
		glog.Infoln("Connected to docker:", this.DockerPort)

	} else {
		docker, err := docker.NewClient(this.DockerPort)
		if err != nil {
			return err
		}
		this.docker = docker
		glog.Infoln("Connected to docker:", this.DockerPort)

	}

	// Set up callbacks
	if this.docker != nil {
		// This is where we get callbacks for the containers that the agent initiates.
		// The event api can include containers that are started manually.
	}

	glog.Infoln("Start Zookeeper events channel")
	go func() {

		status := func(map[string]interface{}) {
			// no-op
		}
		if this.StatusPubsubTopic != "" {

			root, err := template.ApplyTemplate(this.StatusPubsubTopic, this, map[string]interface{}{
				"env": func(p string) *string {
					return zk.GetString(this.zk, registry.Path(p))
				},
				"domain_service": func(p string) *string {
					return zk.GetString(this.zk, registry.Path("/"+this.Domain).Sub(p))
				},
			})
			if err != nil {
				panic(err)
			}

			id := path.Join(this.Domain, this.Name, this.Host)
			topic := pubsub.Topic(root + "/" + id)

			glog.Infoln("STATUS-TOPIC: Status topic=", topic)

			if topic.Valid() {
				if pb, err := topic.Broker().PubSub(id); err == nil {
					status = func(evt map[string]interface{}) {
						msg, err := json.Marshal(evt)
						if err == nil {
							pb.Publish(topic, msg)
						}
					}
					this.statusTopic = topic
					glog.Infoln("STATUS-TOPIC: Status topic=", topic, "ready.")
				}
			} else {
				panic(topic)
			}
		}

		events := this.zk.Events()
		for {
			evt := <-events
			glog.Infoln("ZKEvent:", evt.JSON())

			// send as pubsub
			// TODO - Redpill compatible:
			/*
				type Event struct {
					Status      string `json:"status"`
					Title       string `json:"title,omitempty"`
					Description string `json:"description,omitempty"`
					Note        string `json:"note,omitempty"`
					User        string `json:"user,omitempty"`
					Type        string `json:"type,omitempty"`
					Url         string `json:"url,omitempty"`
					Timestamp   int64  `json:"timestamp,omitempty"`
					ObjectId    string `json:"object_id"`
					ObjectType  string `json:"object_type"`
				}
			*/
			m := evt.AsMap()
			m["object_id"] = path.Join(this.Domain, this.Name, this.Host)
			m["object_type"] = "agent"
			m["timestamp"] = time.Now().Unix()
			m["description"] = fmt.Sprint(m["type"], ":", m["state"], "@", "server=", m["server"])
			m["title"] = "zookeeper event from agent " + path.Join(this.Domain, this.Name, this.Host)
			m["user"] = "dash"
			switch m["state"] {
			case "state-disconnected", "state-auth-failed", "state-expired":
				m["status"] = "fatal"
			case "state-connected", "state-has-session":
				m["status"] = "ok"
			}
			status(m)
		}
	}()

	return nil
}

func (this *Agent) GetInfo() interface{} {
	info := Info{
		Now:         time.Now(),
		Api:         fmt.Sprintf("%s:%d", this.Host, this.ListenPort),
		DockerApi:   fmt.Sprintf("http://%s:%d/dockerapi", this.Host, this.ListenPort),
		Version:     *version.BuildInfo(),
		StatusTopic: this.statusTopic.String(),
		Agent:       this,
		Environ:     os.Environ(),
	}
	if this.is_running_dockerui() {
		info.DockerUi = fmt.Sprintf("http://%s:%d/", this.Host, this.DockerUIPort)
	}
	return info
}

func (this *Agent) Register() error {
	if this.zk == nil {
		return ErrNotConnectedToRegistry
	}
	attempts := 0
	for {
		key := registry.NewPath("dash", this.Host)
		err := zk.CreateOrSet(this.zk, key, this.GetInfo(), true)
		glog.Infoln("Register key=", key, "err=", err)

		if err == nil {
			// Update this only on successful registration
			this.Registration = key.Path()
			return nil
		} else {
			if attempts == 12 {
				return err
			} else {
				time.Sleep(5 * time.Second)
				attempts += 1
			}
		}
	}
}

// Containers in this domain
func (this *Agent) ListContainers(domain, service string) ([]*docker.Container, error) {
	d, has := this.domains[domain]
	if !has {
		return nil, ErrNoDomain
	}
	return d.ListContainers(ServiceKey(service))
}

func (this *Agent) WatchContainer(domain, service string, spec *MatchContainerRule) error {
	d, has := this.domains[domain]
	if !has {
		return ErrNoDomain
	}

	this.lock.Lock()
	defer this.lock.Unlock()
	return d.WatchContainer(ServiceKey(service), spec)
}

func (this *Agent) ConfigureDomain(config *DomainConfig) (*Domain, error) {
	if config.Domain == "" {
		return nil, ErrNoDomain
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	domain, has := this.domains[config.Domain]
	if !has {

		domain = &Domain{
			Domain:                 config.Domain,
			RegistryContainerEntry: config.RegistryContainerEntry,
			zk:                 this.zk,
			docker:             this.docker,
			container_watchers: make(map[string]chan<- bool, 0),
			triggers:           NewZkWatcher(this.zk),
			agent:              this,
			tracker:            NewContainerTracker(config.Domain),
			schedulers:         make(map[ServiceKey]*Scheduler),
		}

		_, err := domain.StartScheduleExecutor()
		if err != nil {
			glog.Warningln("Cannot start schedule executor")
			return nil, err
		}

		domain.Host = this.Host
		domain.Port = this.Port

		err = domain.Register()
		if err != nil {
			return nil, err
		}

		this.domains[config.Domain] = domain
		this.domainConfigs[config.Domain] = *config
		domain.Config = config
	}
	return this.domains[config.Domain], nil
}

func (this *Agent) onMatchContainer(c *docker.Container, match_rule *ContainerMatchRule) {
	// Need to increment a counter regardless of the container's state
	cc := get_sequence_by_image(c.Image)
	glog.V(100).Infoln("Service=", match_rule.Service, "Image=", c.Image, "SequenceCounter=", cc)

	if d, has := this.domains[match_rule.Domain]; has {

		glog.V(100).Infoln("Service=", match_rule.Service, "Id=", c.Id[0:12],
			"FinishedAt=", c.DockerData.State.FinishedAt)

		// Match the name but we need to take into account if it's running or not.
		switch {

		case c.DockerData.State.Restarting:
			d.tracker.Starting(match_rule.Service, c)

		case c.DockerData.State.Running, c.DockerData.State.Restarting:
			d.tracker.Running(match_rule.Service, c)
			glog.Infoln("Registering container Id=", c.Id, "Image=", c.Image, "Rule=", match_rule)
			if entry, err := BuildRegistryEntry(c, match_rule.GetMatchContainerPort()); entry != nil {
				entry.Domain = match_rule.Domain
				entry.Service = string(match_rule.Service)
				entry.Host = this.Host

				err = entry.Register(this.zk)

				if err != nil {
					glog.Warningln("Error during registration:", err)
				}
				k, v, _ := entry.KeyValue()
				glog.Infoln("Registered", k, v)
			} else {
				glog.Warning("Error building registry", err, "for", *c)
			}

		case c.DockerData.State.FinishedAt.Before(time.Now()):
			glog.V(100).Infoln("Container", "Id=", c.Id[0:12], "Name=", c.Name, "stopped.")
			d.tracker.Stopped(match_rule.Service, c)
		}
	}
}
