package dash

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/circleci"
	"github.com/qorio/maestro/pkg/zk"
	"strings"
)

type CircleCi struct {
	ZkSettings
	circleci.Config

	AuthZkPath  string `json:"circle_auth_zkpath"`
	BuildNumber int64  `json:"circle_buildnum"`
	TargetDir   string `json:"target_dir,omitempty"`
}

func (this *CircleCi) Connect() zk.ZK {
	zookeeper, err := zk.Connect(strings.Split(this.Hosts, ","), this.Timeout)
	if err != nil {
		panic(err)
	}
	return zookeeper
}

func (this *CircleCi) Run() error {
	glog.Infoln("CircleCI with config:", this)

	if this.AuthZkPath != "" && this.ApiToken == "" {
		glog.Infoln("Fetching configuration from zk")

		zkc := this.Connect()
		n, err := zkc.Get(this.AuthZkPath)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(n.Value, &this.Config)
		if err != nil {
			return err
		}
	}

	all, err := this.Config.FetchBuildArtifacts(this.BuildNumber, nil)
	if err != nil {
		return err
	}

	for _, a := range all {
		len, err := a.Download(this.TargetDir)
		if err != nil {
			panic(err)
		}
		glog.Infoln("Downloaded", a.Name, "len=", len)
	}
	return nil
}
