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
	"github.com/infradash/dash/pkg/proxy"
	"github.com/infradash/dash/pkg/registry"
	"github.com/infradash/dash/pkg/restart"
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

	restart := &restart.Restart{}
	restart.BindFlags()

	proxy := &proxy.Proxy{}
	proxy.BindFlags()

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

	regContainerEntry.Identity = *identity

	executor.Identity = *identity
	executor.QualifyByTags.Tags = tags
	executor.ZkSettings = *zkSettings
	envSource.RegistryEntryBase = *regEntryBase
	executor.EnvSource = *envSource
	if len(flag.Args()) > 1 {
		executor.Cmd.Path = flag.Args()[1]
	}
	if len(flag.Args()) > 2 {
		executor.Cmd.Args = flag.Args()[2:]
	}

	switch verb {

	case "proxy":
		glog.Infoln(buildInfo.Notice())

		proxy.Initializer = initializer

		// Special argument after 'proxy' is interpreted as the config url
		if len(flag.Args()) > 1 {
			proxy.Initializer.ConfigUrl = flag.Args()[1]
		}
		proxy_done := make(chan error)
		go func() {
			glog.Infoln("Starting proxy:", *proxy)
			proxy_done <- proxy.Run()
		}()

		// Make sure proxy finishes
		err := <-proxy_done
		if err != nil {
			panic(err)
		}

	case "restart":
		glog.Infoln(buildInfo.Notice())

		restart.RegistryReleaseEntry = *regReleaseEntry
		restart.ZkSettings = *zkSettings
		restart.Initializer = initializer

		// Special argument after 'proxy' is interpreted as the config url
		if len(flag.Args()) > 1 {
			restart.Initializer.ConfigUrl = flag.Args()[1]
		}
		restart_done := make(chan error)
		go func() {
			glog.Infoln("Starting restart:", *restart)
			restart_done <- restart.Run()
		}()

		// Make sure restart finishes
		err := <-restart_done
		if err != nil {
			panic(err)
		}

	case "terraform":
		glog.Infoln(buildInfo.Notice())

		// disable the initializer so that it's loaded by terraform instead
		executor.Initializer = nil
		terraform.Executor = *executor

		// start terraform steps in a separate thread
		terraform_done := make(chan error)
		go func() {
			glog.Infoln("Starting terraform CONFIG:", *terraform, terraform.Identity.String(), terraform.Initializer.Context)
			terraform_done <- terraform.Run()
		}()

		glog.Infoln("Starting terraform EXEC:", *terraform, terraform.Identity.String(), terraform.Initializer.Context)
		terraform.Executor.Exec()

		// Make sure terrforming is complete
		err := <-terraform_done
		if err != nil {
			panic(err)
		}

		// now just loop
		if terraform.Executor.Daemon {
			glog.Infoln("Terraform in daemon mode.")
			forever := make(chan error)
			<-forever
		}

	case "exec":
		glog.Infoln(buildInfo.Notice())

		glog.Infoln("Exec:", executor, executor.Identity.String(), executor.Initializer.Context)
		executor.Exec()
		err := executor.Wait()
		if err != nil {
			panic(err)
		}

	case "agent":
		glog.Infoln(buildInfo.Notice())

		agent.RegistryContainerEntry = *regContainerEntry
		agent.QualifyByTags.Tags = tags
		agent.ZkSettings = *zkSettings
		agent.DockerSettings = *dockerSettings
		agent.RegistryContainerEntry.RegistryReleaseEntry = *regReleaseEntry
		agent.RegistryContainerEntry.RegistryReleaseEntry.RegistryEntryBase = *regEntryBase

		glog.Infoln(buildInfo.Notice())

		glog.Infoln("Agent.Name=", agent.RegistryContainerEntry.Identity.Name)
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

		if len(flag.Args()) > 1 {
			circleci.Cmd = flag.Args()[1]
		}
		if len(flag.Args()) > 2 {
			circleci.Args = flag.Args()[2:]
		}

		err := circleci.Run() // blocks
		if err != nil {
			panic(err)
		}
	}

	glog.Infoln("Bye")

}
