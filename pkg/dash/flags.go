package dash

import (
	"flag"
	"github.com/qorio/omni/runtime"
	"os"
	"time"
)

func (this *ZkSettings) BindFlags() {
	flag.StringVar(&this.Hosts, "zookeeper", os.Getenv(EnvZookeeper),
		"Comma-delimited zk host:port, e.g. zk1.infradash.io,zk2.infradash.io,zk3.infradash.io:2181")
	flag.DurationVar(&this.Timeout, "timeout", time.Second, "Connection timeout to zk.")
}

func (this *DockerSettings) BindFlags() {
	flag.StringVar(&this.DockerPort, "docker", os.Getenv(EnvDocker),
		"Docker port, unix or tcp, e.g. unix:///var/run/docker.sock")
	flag.StringVar(&this.Cert, "tlscert", "", "Path to cert for Docker TLS client")
	flag.StringVar(&this.Key, "tlskey", "", "Path to private key for Docker TLS client")
	flag.StringVar(&this.Ca, "tlsca", "", "Path to ca for Docker TLS client")
}

func (this *RegistryEntryBase) BindFlags() {
	flag.StringVar(&this.AuthToken, "auth_token", os.Getenv(EnvAuthToken), "Auth token")
	flag.StringVar(&this.Domain, "domain", os.Getenv(EnvDomain), "Namespace domain (e.g. integration.foo.com)")
	flag.StringVar(&this.Service, "service", os.Getenv(EnvService), "Namespace service (e.g. web_app)")
	flag.StringVar(&this.Version, "version", os.Getenv(EnvVersion), "Namespace version (e.g. v1.1.0)")
	flag.StringVar(&this.Path, "path", os.Getenv(EnvPath), "Namespace path")
}

func (this *RegistryReleaseEntry) BindFlags() {
	flag.StringVar(&this.Image, "image", os.Getenv(EnvImage), "Image (e.g. infradash/infradash-api")
	flag.StringVar(&this.Build, "build", os.Getenv(EnvBuild), "Build (e.g. 34)")
}

func (this *RegistryContainerEntry) BindFlags() {
	flag.StringVar(&this.Name, "name", os.Getenv(EnvDockerName), "Docker name")
	flag.StringVar(&this.Host, "host", os.Getenv(EnvHost), "Hostname")
}

func (this *EnvSource) BindFlags() {
	flag.BoolVar(&this.ReadStdin, "stdin", false, "User stdin for input")
}

func (this *Env) BindFlags() {
	flag.BoolVar(&this.Publish, "publish", false, "True to publish entries to destination path")
	flag.StringVar(&this.Path, "env", "", "Environment path")
	flag.BoolVar(&this.Overwrite, "overwrite", false, "True to overwrite env value during publish")
}

func (this *Registry) BindFlags() {
	flag.BoolVar(&this.Release, "release", false, "True to publish release record")
	flag.BoolVar(&this.Setlive, "setlive", false, "True to update record")
	flag.BoolVar(&this.SetliveNoWait, "setlive_nowait", false, "True to not wait")
	flag.BoolVar(&this.Commit, "commit", false, "True to commit the record")
	flag.BoolVar(&this.ReadValue, "read", false, "True to read value from registry")
	flag.StringVar(&this.ReadValuePath, "readpath", "", "The path to read value from")
	flag.IntVar(&this.SetliveMinThreshold, "setlive_min_instances", 1, "Setlive: minimal threshold of available instances before setlive.")
	flag.DurationVar(&this.SetliveWait, "setlive_wait", time.Duration(1*time.Minute), "Setlive: wait internval to check available instances.")
	flag.DurationVar(&this.SetliveMaxWait, "setlive_maxwait", time.Duration(5*time.Minute), "Setlive: max wait before giving up.")

	flag.StringVar(&this.WriteValue, "writevalue", "", "The value to write")
	flag.StringVar(&this.WriteValuePath, "writepath", "", "The path to write")

	flag.IntVar(&this.Retries, "retries", 5, "Retries")
	flag.IntVar(&this.RetriesWaitSeconds, "retries_wait_seconds", 5, "Wait seconds between retries")

	flag.StringVar(&this.SchedulerTriggerPath, "scheduler_trigger_path", "", "Scheduler trigger path; value is a counter")
	flag.StringVar(&this.SchedulerImagePath, "scheduler_image_path", "", "Scheduler image path; value is the image.")
}

func (this *ConfigLoader) BindFlags() {
	flag.StringVar(&this.ConfigUrl, "config_source_url", os.Getenv(EnvConfigUrl), "Initialize config source url")
}

func (this *CircleCi) BindFlags() {
	flag.StringVar(&this.User, "circle_user", "", "Circle user")
	flag.StringVar(&this.Project, "circle_project", "", "Circle project")
	flag.StringVar(&this.ApiToken, "circle_token", "", "Circle token")
	flag.Int64Var(&this.BuildNumber, "circle_buildnum", 0, "Circle build number")
	flag.StringVar(&this.TargetDir, "build_artifact_dir", runtime.EnvString("PWD", "."), "Target directory")
	flag.StringVar(&this.AuthZkPath, "circle_auth_zkpath", "", "Circle Auth zk path")
}
