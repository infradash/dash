package dash

import (
	"time"
)

const (
	EnvAuthToken       = "DASH_AUTH_TOKEN"
	EnvConfigUrl = "DASH_CONFIG_URL"

	EnvZookeeper = "ZOOKEEPER_HOSTS"
	EnvDocker    = "DOCKER_PORT"

	EnvDomain  = "DASH_DOMAIN"
	EnvService = "DASH_SERVICE"
	EnvVersion = "DASH_VERSION"
	EnvPath    = "DASH_PATH"
	EnvTags    = "DASH_TAGS"

	EnvImage = "DASH_IMAGE"
	EnvBuild = "DASH_BUILD"

	EnvHost       = "DASH_HOST"
	EnvDockerName = "DASH_DOCKER_NAME"
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
