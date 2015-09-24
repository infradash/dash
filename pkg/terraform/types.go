package terraform

import (
	"github.com/qorio/maestro/pkg/pubsub"
	"net/url"
)

type Ip string
type Url string

type Server struct {
	Ip       Ip   `json:"ip"`
	Port     int  `json:"port"`
	Observer bool `json:"observer"`
}

type Config struct {
	Template Url      `json:"template"`
	Endpoint Url      `json:"endpoint"`
	Cmd      []string `json:"cmd"`
	Applied  string   `json:"applied"`
}

type TerraformConfig struct {
	Status    pubsub.Topic `json:"status"`
	Ensemble  []Server     `json:"ensemble"`
	Zookeeper *Config      `json:"zookeeper"`
	Kafka     *Config      `json:"kafka"`
}

func (this Url) String() string {
	return string(this)
}

func check_url(s string) error {
	_, err := url.Parse(s)
	return err
}

func (this *TerraformConfig) Validate() error {
	if len(this.Ensemble)%2 == 0 {
		return ErrBadConfig
	}

	if this.Zookeeper != nil {
		if err := this.Zookeeper.Validate(); err != nil {
			return err
		}
	}
	if this.Kafka != nil {
		if err := this.Kafka.Validate(); err != nil {
			return err
		}
	}
	return nil
}
