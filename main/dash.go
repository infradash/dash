package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/infradash/dash/pkg/agent"
	"github.com/infradash/dash/pkg/circleci"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/infradash/dash/pkg/env"
	"github.com/infradash/dash/pkg/executor"
	"github.com/infradash/dash/pkg/registry"
	"github.com/infradash/dash/pkg/terraform"
	"github.com/qorio/omni/version"
	"os"
	"strings"
)

func get_envs() map[string]interface{} {
	envs := make(map[string]interface{})
	for _, kv := range os.Environ() {
		p := strings.Split(kv, "=")
		envs[p[0]] = p[1]
	}
	return envs
}

var (
	TagsList = flag.String("tags", os.Getenv(EnvTags), "Tags for this instance")
)

func main() {

	buildInfo := version.BuildInfo()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n", buildInfo.Notice())
		fmt.Fprintf(os.Stderr, "flags:\n")
		flag.PrintDefaults()
	}

	identity := &Identity{}
	identity.Init()
	identity.BindFlags()

	zkSettings := &ZkSettings{}
	zkSettings.BindFlags()

	dockerSettings := &DockerSettings{}
	dockerSettings.BindFlags()

	regEntryBase := &RegistryEntryBase{}
	regEntryBase.BindFlags()

	envSource := &EnvSource{}
	envSource.BindFlags()

	regContainerEntry := &RegistryContainerEntry{}
	regContainerEntry.BindFlags()

	env := &env.Env{}
	env.BindFlags()

	regReleaseEntry := &RegistryReleaseEntry{}
	regReleaseEntry.BindFlags()

	registry := &registry.Registry{}
	registry.BindFlags()

	initializer := &ConfigLoader{Context: MergeMaps(get_envs(), EscapeVars(ConfigVariables...))}
	initializer.BindFlags()

	agent := &agent.Agent{Initializer: initializer}
	agent.BindFlags()

	executor := &executor.Executor{Initializer: initializer}
	executor.BindFlags()

	circleci := &circleci.CircleCi{}
	circleci.BindFlags()

	terraform := &terraform.Terraform{Initializer: initializer}
	terraform.BindFlags()

	flag.Parse()

	tags := strings.Split(*TagsList, ",")

	if len(flag.Args()) == 0 {
		glog.Infoln("Done")
		os.Exit(0)
	}

	if len(flag.Args()) == 0 {
		panic(errors.New("no-verb"))
	}

	verb := flag.Args()[0]

	switch verb {
	case "terraform":

		terraform.Identity = *identity

		glog.Infoln(buildInfo.Notice())
		glog.Infoln("Starting terraform:", *terraform, terraform.Identity.String(), terraform.Initializer.Context)

		err := terraform.Run()
		if err != nil {
			panic(err)
		}

	case "exec":

		executor.Identity = *identity
		executor.QualifyByTags.Tags = tags
		executor.ZkSettings = *zkSettings
		envSource.RegistryEntryBase = *regEntryBase
		executor.EnvSource = *envSource

		if len(flag.Args()) > 1 {
			executor.Cmd = flag.Args()[1]
		}
		if len(flag.Args()) > 2 {
			executor.Args = flag.Args()[2:]
		}

		glog.Infoln(buildInfo.Notice())
		glog.Infoln("Exec:", executor, executor.Identity.String(), executor.Initializer.Context)
		err := executor.Exec()
		if err != nil {
			panic(err)
		}

	case "agent":

		agent.Identity = *identity
		agent.QualifyByTags.Tags = tags
		agent.ZkSettings = *zkSettings
		agent.DockerSettings = *dockerSettings
		agent.RegistryContainerEntry = *regContainerEntry
		agent.RegistryContainerEntry.RegistryReleaseEntry = *regReleaseEntry
		agent.RegistryContainerEntry.RegistryReleaseEntry.RegistryEntryBase = *regEntryBase

		glog.Infoln(buildInfo.Notice())
		glog.Infoln("Starting agent:", *agent, agent.Identity.String(), agent.Initializer.Context)

		agent.Run() // blocks

	case "env":

		env.ZkSettings = *zkSettings
		env.EnvSource = *envSource
		env.RegistryEntryBase = *regEntryBase // flags for output destination

		err := env.Run() // blocks
		if err != nil {
			panic(err)
		}

	case "registry":

		registry.ZkSettings = *zkSettings
		registry.RegistryReleaseEntry = *regReleaseEntry
		registry.RegistryReleaseEntry.RegistryEntryBase = *regEntryBase

		err := registry.Run()
		if err != nil {
			panic(err)
		}

	case "circleci":

		circleci.ZkSettings = *zkSettings
		err := circleci.Run() // blocks
		if err != nil {
			panic(err)
		}
	}

	glog.Infoln("Bye")

}
