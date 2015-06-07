package dash

import (
	"time"
)

const (
	EnvZookeeper = "ZOOKEEPER_HOSTS"
	EnvDocker    = "DOCKER_PORT"

	EnvDomain     = "DASH_DOMAIN"
	EnvHost       = "DASH_HOST"
	EnvDockerName = "DASH_DOCKER_NAME"
	EnvTags       = "DASH_TAGS"
)

var ConfigVariables = []string{
	"Domain", "Service", "Version", "Repo", "Image", "Tag", "Build", "Running", "Step", "Sequence",
}

type ZkSettings struct {
	Hosts   string        `json:"zk_hosts"`
	Timeout time.Duration `json:"zk_timeout"`
}

type DockerSettings struct {
	DockerPort string `json:"docker_port"`
	Cert       string `json:"cert_path"`
	Key        string `json:"key_path"`
	Ca         string `json:"ca_path"`
}

type QualifyByTags struct {
	Tags []string `json:"tags,omitempty"`
}
