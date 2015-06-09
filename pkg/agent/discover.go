package agent

import (
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/docker"
	"regexp"
	"strings"
)

func (this *Agent) DiscoverSelfInDocker() error {

	docker_name := this.Name

	if docker_name == "" {
		return ErrNoDockerName
	}

	glog.Infoln("Discovering container in Docker with name=", docker_name)

	dc, err := this.docker.FindContainersByName(docker_name)
	if err != nil {
		return err
	}

	if len(dc) > 1 {
		// This is unlikely, since Docker itself will prevent containers with
		// the same assigned name from starting.  However, it's possible that
		// the name provided isn't a complete name, so we have partial match.
		// In this case, it's programming error.
		return ErrMoreThanOneAgent
	}

	if len(dc) == 1 {
		this.self_container = dc[0]

		this.self_container.Inspect()
		glog.Infoln("Found self as container:", this.self_container)

		if len(dc[0].Ports) > 0 {
			this.Port = dc[0].Ports[0]
		}
	}

	return nil
}

type ContainerMatchRule struct {
	WatchContainerSpec

	Domain  string
	Service ServiceKey
}

func (this *WatchContainerSpec) GetMatchContainerPort() int {
	if this.MatchContainerPort != nil {
		return *this.MatchContainerPort
	} else {
		return 0
	}
}

func (this *ContainerMatchRule) GetMatchContainerPort() int {
	return this.WatchContainerSpec.GetMatchContainerPort()
}

func (this *ContainerMatchRule) match_by_environment(c *docker.Container) bool {
	if len(this.MatchContainerEnvironment) == 0 {
		return true // Don't care
	}

	// want to match, but have no environments defined:
	if c.DockerData == nil || c.DockerData.Config == nil || len(c.DockerData.Config.Env) == 0 {
		return false
	}

	// name=value (regexp) pairs separated by comma ==> AND'ing
	to_match := map[string]string{}
	for _, nv := range this.MatchContainerEnvironment {
		to_match[nv] = nv
	}

	for _, env := range c.DockerData.Config.Env {
		for k, pattern := range to_match {
			// use the full nv (e.g. FOO=BAR) as regexp
			if match, _ := regexp.MatchString(pattern, env); match {
				delete(to_match, k) // remove it so we don't match again
				break
			}
		}
	}
	return len(to_match) == 0
}

func (this *ContainerMatchRule) match_by_name_regexp(c *docker.Container) bool {
	if this.MatchContainerName == nil {
		return true // Don't care
	}
	regex := *this.MatchContainerName
	m, _ := regexp.MatchString(regex, c.Name)
	return m
}

func (this *ContainerMatchRule) match_by_live_port(c *docker.Container) bool {
	if this.MatchContainerPort == nil {
		return true // Don't care
	}
	port := *this.MatchContainerPort
	if port > 0 {
		for _, p := range c.Ports {
			if port == int(p.ContainerPort) {
				return true
			}
		}
	}
	return false
}

func (this *ContainerMatchRule) match(c *docker.Container) bool {
	return ImageMatch(c.Image, &this.Image) &&
		this.match_by_name_regexp(c) && this.match_by_live_port(c) && this.match_by_environment(c)
}

type CheckContainer func(*docker.Container) (bool, *ContainerMatchRule)
type OnMatch func(*docker.Container, *ContainerMatchRule)

func (this *Agent) DiscoverRunningContainers(check CheckContainer, do OnMatch) error {

	glog.Infoln("Querying docker for all containers")
	all_containers, err := this.docker.FindContainers(nil) //map[string][]string{"status": []string{"running"}})
	if err != nil {
		return err
	}
	glog.Infoln("Found", len(all_containers), "containers")

	for _, container := range all_containers {
		glog.V(100).Infoln("Checking", "Name=", container.Name, "Image=", container.Image, "Id=", container.Id[0:12])
		if match, match_rule := check(container); match {
			glog.V(100).Infoln("Matched", "Name=", container.Name, "Id=", container.Id[0:12],
				"Image=", container.Image, "Service=", match_rule.Service)
			do(container, match_rule)
		}
	}
	return nil
}

type DiscoveryContainerMatcher struct {
	imagesByDomain map[string]map[ServiceKey]ContainerMatchRule
}

func (this *DiscoveryContainerMatcher) Init() *DiscoveryContainerMatcher {
	this.imagesByDomain = make(map[string]map[ServiceKey]ContainerMatchRule)
	return this
}

func (this *DiscoveryContainerMatcher) C(domain string, service ServiceKey, spec *WatchContainerSpec) *DiscoveryContainerMatcher {
	match_rule := ContainerMatchRule{
		WatchContainerSpec: *spec,
		Domain:             domain,
		Service:            service,
	}
	if _, has := this.imagesByDomain[domain]; !has {
		this.imagesByDomain[domain] = map[ServiceKey]ContainerMatchRule{service: match_rule}
	} else {
		this.imagesByDomain[domain][service] = match_rule
	}
	return this
}

func findContainerDomain(c *docker.Container) *string {
	if c.DockerData == nil || c.DockerData.Config == nil {
		return nil
	}

	v := ""
	// First we check to see if the docker container was started with a particular environment
	search := fmt.Sprintf("%s=", EnvDomain)
	if c.DockerData != nil && c.DockerData.Config != nil {
		for _, env := range c.DockerData.Config.Env {
			index := strings.Index(env, search)
			if index == 0 {
				v = env[len(search):]
				break
			}
		}
	}
	return &v
}

func (this *DiscoveryContainerMatcher) MatcherForDomain(domain string, service ServiceKey) func(docker.Action, *docker.Container) bool {
	// get the rule
	if service_rule_map, has_domain := this.imagesByDomain[domain]; has_domain {
		if rule, has_map := service_rule_map[service]; has_map {
			return func(a docker.Action, c *docker.Container) bool {
				if a == docker.Remove {
					return ImageMatch(c.Image, &rule.Image)
				}

				if env := findContainerDomain(c); env != nil {
					return *env == domain && rule.match(c)
				} else {
					return rule.match(c)
				}
			}
		}
	}
	glog.Warningln("No matcher for Domain=", domain, "Service=", service)
	return func(docker.Action, *docker.Container) bool {
		return false
	}
}
func (this *DiscoveryContainerMatcher) match(domain *string, c *docker.Container) (bool, *ContainerMatchRule) {
	if domain != nil {
		// Now we have matched by the domain of the container.  Let's see if it's running an image we care about:
		for _, match_rule := range this.imagesByDomain[*domain] {
			if env := findContainerDomain(c); env != nil {
				if *env == *domain && match_rule.match(c) {
					return true, &match_rule
				}
			} else if match_rule.match(c) {
				return true, &match_rule
			}
		}
		return false, nil
	} else {
		// if we don't know the domain, then search through all the images...
		for _, rules := range this.imagesByDomain {
			for _, match_rule := range rules {
				if match_rule.match(c) {
					return true, &match_rule
				}
			}
		}
		return false, nil
	}

}

func (this *DiscoveryContainerMatcher) Match(c *docker.Container) (bool, *ContainerMatchRule) {
	return this.match(findContainerDomain(c), c)
}

func ImageMatch(image string, spec *docker.Image) bool {
	if spec.Tag != "" {
		return image == fmt.Sprintf("%s:%s", spec.Repository, spec.Tag)
	} else {
		return strings.Index(image, spec.Repository) == 0
	}
}

func BuildRegistryEntry(container *docker.Container, match_port int) (*RegistryContainerEntry, error) {
	_, version, _, err := ParseVersion(container.Image)
	if err != nil {
		return nil, err
	}

	entry := &RegistryContainerEntry{
		RegistryReleaseEntry: RegistryReleaseEntry{
			RegistryEntryBase: RegistryEntryBase{
				Version: version,
			},
			Image: container.Image,
		},
		ContainerId: container.Id,
	}

	// Interactive sessions don't have ports... So we add an entry only if we don't care
	// to match ports (match_container_port == 0)
	for _, p := range container.Ports {
		if match_port == int(p.ContainerPort) || match_port == 0 {
			entry.Port = p
			return entry, nil
		}
	}

	if match_port == 0 {
		return entry, nil
	} else {
		return nil, nil
	}
}
