package dash

import (
	"fmt"
	"github.com/qorio/omni/common"
	"time"
)

const (
	EnvAuthToken = "DASH_AUTH_TOKEN"
	EnvConfigUrl = "DASH_CONFIG_URL"
	EnvZkHosts   = "DASH_ZK_HOSTS"
	EnvDocker    = "DASH_DOCKER_PORT"
	EnvDomain    = "DASH_DOMAIN"
	EnvService   = "DASH_SERVICE"
	EnvVersion   = "DASH_VERSION"
	EnvPath      = "DASH_PATH"
	EnvTags      = "DASH_TAGS"
	EnvName      = "DASH_NAME"
	EnvHost      = "DASH_HOST"
	EnvImage     = "DASH_IMAGE"
	EnvBuild     = "DASH_BUILD"
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

type Identity struct {
	Id           string `json:"id"`
	Name         string `json:"name,omitemtpy"`
	Registration string `json:"registration,omitempty"`
}

func (this *Identity) Init() {
	this.Id = common.NewUUID().String()
}

func (this *Identity) String() string {
	s := fmt.Sprintf("%s/%s", this.Name, this.Id)
	if this.Registration != "" {
		return s + fmt.Sprintf("[%s]", this.Registration)
	} else {
		return s
	}
}
