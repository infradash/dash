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
	Template            Url `json:"template"`
	Endpoint            Url `json:"endpoint"`
	CheckStatusEndpoint Url `json:"check_status_endpoint"`

	Stop chan bool
}

type ZookeeperConfig Config
type KafkaConfig Config

type TerraformConfig struct {
	Status    pubsub.Topic     `json:"status"`
	Ensemble  []Server         `json:"ensemble"`
	Zookeeper *ZookeeperConfig `json:"zookeeper"`
	Kafka     *KafkaConfig     `json:"kafka"`
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
