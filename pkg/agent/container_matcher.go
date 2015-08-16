package agent

import (
	"github.com/qorio/maestro/pkg/docker"
	"regexp"
)

func (this *ContainerMatchRulesUnion) Match(c *docker.Container) bool {
	return this.match_by_labels(c) && this.match_by_environment(c) && this.match_by_name(c)
}

func (this *ContainerMatchRulesUnion) match_by_labels(c *docker.Container) bool {
	if len(this.ByContainerLabels) == 0 {
		return true // Don't care
	}

	// want to match, but have no environments defined:
	if c.DockerData == nil || c.DockerData.Config == nil || len(c.DockerData.Config.Labels) == 0 {
		return false
	}

	k, m := 0, 0
	for k, pattern := range this.ByContainerLabels {
		if match, _ := regexp.MatchString(pattern, c.DockerData.Config.Labels[k]); match {
			m += 1
		}
	}
	return m == k+1
}

func (this *ContainerMatchRulesUnion) match_by_environment(c *docker.Container) bool {
	if len(this.ByContainerEnvironment) == 0 {
		return true // Don't care
	}

	// want to match, but have no environments defined:
	if c.DockerData == nil || c.DockerData.Config == nil || len(c.DockerData.Config.Env) == 0 {
		return false
	}

	// name=value (regexp) pairs separated by comma ==> AND'ing
	to_match := map[string]string{}
	for _, nv := range this.ByContainerEnvironment {
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

func (this *ContainerMatchRulesUnion) match_by_name(c *docker.Container) bool {
	if this.ByContainerName == nil {
		return true // Don't care
	}
	regex := *this.ByContainerName
	m, _ := regexp.MatchString(regex, c.Name)
	return m
}
