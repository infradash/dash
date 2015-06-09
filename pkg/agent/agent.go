package agent

import (
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
	"github.com/qorio/maestro/pkg/zk"
	"github.com/qorio/omni/runtime"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Agent struct {
	QualifyByTags

	ZkSettings
	DockerSettings

	RegistryContainerEntry

	Identity   string `json:"identity"`
	ListenPort int    `json:"listen_port"`

	UiDocRoot string `json:"ui_doc_root,omitempty"`
	EnableUI  bool   `json:"enable_ui,omitempty"`

	Initializer *ConfigLoader `json:"config_loader"`

	selfRegister bool `json:"-"`

	// json skips these fields
	endpoint       http.Handler      `json:"-"`
	zk             zk.ZK             `json:"-"`
	docker         *docker.Docker    `json:"-"`
	self_container *docker.Container `json:"-"`

	lock          sync.Mutex
	domains       map[string]*Domain
	domainConfigs map[string]DomainConfig

	//containerMatcher *DiscoveryContainerMatcher
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

// Block until SIGTERM
func (this *Agent) Run() {

	glog.Infoln("Agent", this.GetIdentity())

	if this.EnableUI && this.UiDocRoot != "" {
		mux := http.NewServeMux()
		glog.Infoln("Starting UI with docroot=", this.UiDocRoot, "DockerPort=", this.DockerPort)
		fileHandler := http.FileServer(http.Dir(this.UiDocRoot))
		// Proxy to docker -- the Docker API handler
		dockerApiHandler := this.createDockerApiHandler(this.UiDocRoot, this.DockerPort)
		mux.Handle("/dockerapi/", http.StripPrefix("/dockerapi", dockerApiHandler))
		mux.Handle("/", fileHandler)

		go func() {
			p := fmt.Sprintf(":%d", this.ListenPort+1)
			glog.Infoln("UI Listening on", p)
			if err := http.ListenAndServe(p, mux); err != nil {
				panic(err)
			}
		}()
	}

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

	if this.Initializer != nil {
		glog.Infoln("Loading configuration from", this.Initializer.SourceUrl)

		var list []DomainConfig
		err := this.Initializer.Load(&list)
		if err != nil {
			panic(err)
		}

		applied_domain_configs := []*DomainConfig{}

		for _, per_domain := range list {

			applied := new(DomainConfig)
			err := ApplyVarSubs(per_domain, applied, MergeMaps(map[string]interface{}{
				"Domain": per_domain.Domain,
			}, EscapeVars(ConfigVariables[1:]...)))
			if err != nil {
				panic(err)
			}
			per_domain = *applied
			applied_domain_configs = append(applied_domain_configs, &per_domain)

			// for svc, watchContainerSpec := range per_domain.WatchContainers {
			// 	if watchContainerSpec.QualifyByTags.Matches(this.QualifyByTags.Tags) {
			// 		matcher.C(per_domain.Domain, svc, watchContainerSpec)
			// 	}
			// }

			if _, config_err := this.ConfigureDomain(&per_domain); config_err != nil {
				panic(config_err)
			}
		}

		glog.Infoln("Start running discovery")
		matcher := new(DiscoveryContainerMatcher).Init()
		for _, domain := range this.domains {
			watches, err := domain.GetContainerWatcherSpecs()
			if err != nil {
				panic(err)
			}
			for svc, watch := range watches {
				matcher.C(domain.Domain, svc, watch)
			}
		}

		err = this.DiscoverRunningContainers(
			matcher.Match,
			func(c *docker.Container, match_rule *ContainerMatchRule) {

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

						glog.Infoln("Registering container Id=", c.Id, "Image=", c.Image)
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
			})
		if err != nil {
			glog.Infoln("Error discovering containers:", err)
		}

		// Configure and start up services
		glog.Infoln("Configure domains")
		for _, domain := range this.domains {
			glog.Infoln("Starting services: Domain=", domain.Domain)
			_, config_err := domain.StartServices(this.QualifyByTags)
			if config_err != nil {
				panic(config_err)
			}
		}

		glog.Infoln("Synchronize local states with scheduler")
		for _, domain := range this.domains {
			err := domain.SynchronizeSchedule()
			if err != nil {
				glog.Warningln("Failed to synchronize scheduling for Domain=", domain.Identity, "Err=", err)
			}
		}
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
	zk, err := zk.Connect(strings.Split(this.Hosts, ","), this.Timeout)
	if err != nil {
		return err
	}
	this.zk = zk
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
	return nil
}

func (this *Agent) Register() error {
	attempts := 0
	for {
		err := this._register()
		if err != nil {
			if attempts == 12 {
				return err
			} else {
				time.Sleep(5 * time.Second)
				attempts += 1
			}
		} else {
			return nil
		}
	}
}

func (this *Agent) _register() error {
	if this.zk == nil {
		return ErrNotConnectedToRegistry
	}

	if this.self_container != nil {
		key, value, err := RegistryKeyValue(KAgent, this)
		glog.Infoln("Register self as key=", key, "value=", value, "err=", err)
		if err != nil {
			return err
		}
		n, err := this.zk.CreateEphemeral(key, nil)
		if err != nil {
			return err
		} else {
			err = n.Set([]byte(value))
			if err == nil {
				// Update this only on successful registration
				this.Identity = key
			}
		}
	} else {
		return ErrNoContainerInformation
	}

	return nil
}

// Containers in this domain
func (this *Agent) ListContainers(domain, service string) ([]*docker.Container, error) {
	d, has := this.domains[domain]
	if !has {
		return nil, ErrNoDomain
	}
	return d.ListContainers(ServiceKey(service))
}

func (this *Agent) WatchContainer(domain, service string, spec *WatchContainerSpec) error {
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
