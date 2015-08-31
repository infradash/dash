package circleci

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/circleci"
	"github.com/qorio/maestro/pkg/zk"
	"io"
	"os"
	"strings"
	"time"
)

var (
	ErrNoCircleYml = errors.New("error-no-circle-yml")
)

type CircleCi struct {
	ZkSettings
	circleci.Build

	AuthZkPath string `json:"circle_auth_zkpath"`

	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

func (this *CircleCi) Connect() zk.ZK {
	zookeeper, err := zk.Connect(strings.Split(this.Hosts, ","), this.Timeout)
	if err != nil {
		panic(err)
	}
	return zookeeper
}

func (this *CircleCi) logStart(p circleci.Phase) {
	desc := fmt.Sprint("Stating build ", this.BuildNum, " for project ", this.Project, " by ", this.User)
	fmt.Printf("****,[,%d,Start of %s,%s\n", time.Now().Unix(), p, desc)
}

func (this *CircleCi) logEnd(p circleci.Phase, err error) bool {
	state := "****"
	desc := fmt.Sprint(p, " completed.")
	if err != nil {
		state = "!!!!"
		desc = fmt.Sprint("Build ", this.BuildNum, " failed with error: ", err.Error())
	}
	fmt.Printf("%s,],%d,End of %s,%s\n", state, time.Now().Unix(), p, desc)
	return err == nil
}

func (this *CircleCi) Run() error {

	this.LogStart = this.logStart
	this.LogEnd = this.logEnd

	glog.Infoln("CircleCI with config:", this)

	if this.AuthZkPath != "" && this.ApiToken == "" {
		glog.Infoln("Fetching configuration from zk")

		zkc := this.Connect()
		n, err := zkc.Get(this.AuthZkPath)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(n.Value, &this.Build)
		if err != nil {
			return err
		}
	}

	switch this.Cmd {
	case "fetch", "": // empty string is legacy
		return this.FetchBuildArtifacts()
	case "build":
		if len(this.Args) == 0 {
			return ErrNoCircleYml
		}
		yml, err := os.Open(this.Args[0])
		if err != nil {
			return err
		}
		defer yml.Close()
		return this.RunCircleYml(yml)
	}
	return nil
}

func (this *CircleCi) RunCircleYml(r io.Reader) error {
	glog.Infoln("Running circle build")
	yml := new(circleci.CircleYml)
	err := yml.LoadFromReader(r)
	if err != nil {
		return err
	}
	buff, err := yml.AsYml()
	if err != nil {
		return err
	}
	glog.Infoln("Running\n", string(buff))
	err = this.Build.Build(yml)
	glog.Infoln("Err=", err)
	return err
}

func (this *CircleCi) FetchBuildArtifacts() error {
	all, err := this.Build.FetchBuildArtifacts(this.BuildNum, nil)
	if err != nil {
		return err
	}

	for _, a := range all {
		len, err := a.Download(this.ArtifactsDir)
		if err != nil {
			panic(err)
		}
		glog.Infoln("Downloaded", a.Name, "len=", len)
	}
	return nil
}
