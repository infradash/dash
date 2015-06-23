package dash

import (
	"flag"
	"os"
	"time"
)

func (this *Identity) BindFlags() {
	flag.StringVar(&this.Id, "id", os.Getenv(EnvId), "Id")
	flag.StringVar(&this.Name, "name", os.Getenv(EnvName), "Name")
	flag.StringVar(&this.AuthToken, "auth_token", os.Getenv(EnvAuthToken), "Auth token")
}

func (this *ZkSettings) BindFlags() {
	flag.StringVar(&this.Hosts, "zk_hosts", os.Getenv(EnvZkHosts), "Comma-delimited host:port, e.g. host1:2181,host2:2181")
	flag.DurationVar(&this.Timeout, "timeout", time.Second, "Connection timeout to zk.")
}

func (this *DockerSettings) BindFlags() {
	flag.StringVar(&this.DockerPort, "docker", os.Getenv(EnvDocker), "Docker port, e.g. unix:///var/run/docker.sock")
	flag.StringVar(&this.Cert, "tlscert", "", "Path to cert for Docker TLS client")
	flag.StringVar(&this.Key, "tlskey", "", "Path to private key for Docker TLS client")
	flag.StringVar(&this.Ca, "tlsca", "", "Path to ca for Docker TLS client")
}

func (this *ConfigLoader) BindFlags() {
	flag.StringVar(&this.ConfigUrl, "config_url", os.Getenv(EnvConfigUrl), "Initialize config source url")
}

func (this *RegistryEntryBase) BindFlags() {
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
	flag.StringVar(&this.Host, "host", os.Getenv(EnvHost), "Hostname")
}

func (this *EnvSource) BindFlags() {
	flag.StringVar(&this.Url, "url", "", "Url to source env from")
}
