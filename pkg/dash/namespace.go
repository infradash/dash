package dash

import (
	"bytes"
	"errors"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/zk"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	KReleaseWatch = `
{{define "KEY"}}/{{.Domain}}/{{.Service}}{{end}}
{{define "VALUE"}}/{{.Domain}}/{{.Service}}/{{.Version}}{{end}}
`
	KRelease = `
{{define "KEY"}}/{{.Domain}}/{{.Service}}/{{.Version}}{{end}}
{{define "VALUE"}}{{.Image}}{{end}}
`
	KEnvRoot = `
{{define "KEY"}}/{{.Domain}}/{{.Service}}/{{.Version}}/env{{end}}
{{define "VALUE"}}{{end}}
`
	KEnv = `
{{define "KEY"}}/{{.Domain}}/{{.Service}}/{{.Version}}/env/{{.EnvName}}{{end}}
{{define "VALUE"}}{{.EnvValue}}{{end}}
`
	KImage = `
{{define "KEY"}}/{{.Domain}}/{{.Service}}/{{.Version}}/container/{{.Image}}{{end}}
{{define "VALUE"}}{{.Count}}{{end}}
`
	KContainer = `
{{define "KEY"}}/{{.Domain}}/{{.Service}}/{{.Version}}/container/{{.Image}}/{{.ContainerId}}:{{.ContainerPort}}{{end}}
{{define "VALUE"}}{{.Host}}:{{.HostPort}}{{end}}
`

	// Live watch node and information nodes are separate.  This is so we can implement a 'touch'
	// functionality - the counter value increases while the Live node has the actual version information
	KLiveWatch = `
{{define "KEY"}}/{{.Domain}}/{{.Service}}/live/watch{{end}}
{{define "VALUE"}}0{{end}}
`
	KLive = `
{{define "KEY"}}/{{.Domain}}/{{.Service}}/live{{end}}
{{define "VALUE"}}/{{.Domain}}/{{.Service}}/{{.Version}}/container/{{.Image}},/{{.Domain}}/{{.Service}}/{{.Version}}/env{{end}}
`
	KDash = `
{{define "KEY"}}/{{.Domain}}/dash/{{.Host}}:{{.ContainerPort}}:{{.HostPort}}{{end}}
{{define "VALUE"}}{{.Host}}:{{.HostPort}}{{end}}
`
	KAgent = `
{{define "KEY"}}/dash/{{.Host}}:{{.ContainerPort}}:{{.HostPort}}{{end}}
{{define "VALUE"}}{{.Host}}:{{.HostPort}}{{end}}
`
)

var templates = make(map[string]*template.Template)

func init() {
	must_compile_template(KRelease)
	must_compile_template(KReleaseWatch)
	must_compile_template(KImage)
	must_compile_template(KContainer)
	must_compile_template(KEnvRoot)
	must_compile_template(KEnv)
	must_compile_template(KDash)
	must_compile_template(KLive)
	must_compile_template(KLiveWatch)
	must_compile_template(KAgent)
}

func must_compile_template(k string) {
	templates[k] = template.Must(template.New(k).Parse(k))
	if templates[k].Lookup("KEY") == nil {
		panic("bad-template-definition-no-key:" + k)
	}
	if templates[k].Lookup("VALUE") == nil {
		panic("bad-template-definition-no-value:" + k)
	}
}

func RegistryKeyValue(k string, object interface{}) (key, value string, err error) {
	var keyBuff, valueBuff bytes.Buffer
	err = templates[k].Lookup("KEY").Execute(&keyBuff, object)
	if err != nil {
		return
	}
	err = templates[k].Lookup("VALUE").Execute(&valueBuff, object)
	if err != nil {
		return
	}
	return keyBuff.String(), valueBuff.String(), nil
}

func ParseDockerImage(dockerImage string) (repo, tag string, err error) {
	// docker image name := repo:tag tag := version-build
	i := strings.Index(dockerImage, ":")
	if i > -1 {
		repo = dockerImage[:i]
		tag = dockerImage[i+1:]
		return
	} else {
		err = errors.New("bad image:" + dockerImage)
		return
	}
}

func ParseVersion(dockerImage string) (repo, version, build string, err error) {
	// docker image name := repo:tag tag := version-build
	i := strings.Index(dockerImage, ":")
	if i > -1 {
		repo = dockerImage[:i]
		tag := dockerImage[i+1:]
		j := strings.LastIndex(tag, "-")
		if j > -1 {
			version = tag[:j]
			build = tag[j+1:]
			return
		} else {
			version = tag
			build = ""
			return
		}
	}
	return "", "", "", errors.New("bad image name:" + dockerImage)
}

func ParseLiveValue(value string) (container_path, environment_path string) {
	parts := strings.Split(value, ",")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func ParseHostPort(value string) (host, port string) {
	parts := strings.Split(value, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func (this RegistryContainerEntry) KeyValue() (string, string, error) {
	return RegistryKeyValue(KContainer, this)
}

func (this RegistryContainerEntry) Register(zk zk.ZK) error {
	regkey, value, err := this.KeyValue()
	if err != nil {
		return err
	}
	_, err = zk.CreateEphemeral(regkey, []byte(value))
	if err != nil {
		return err
	}
	return err
}

// When a container is stopped or removed, the port information will be gone.
// Since we use the service port as part of the registry key, we won't be able
// to recover the correct registry key.  Instead, we just take the container id
// and scan all the children and delete those entries with the container id.
// This is a better implementation as it will allow the agent to clean up the entries
// correctly: a container can have multiple service ports and all those will be remov
func (this RegistryContainerEntry) Remove(zkc zk.ZK) error {
	regkey, _, err := RegistryKeyValue(KContainer, this)
	if err != nil {
		return err
	}

	// We use this regkey to figure out the parent node:
	containers_path := filepath.Dir(regkey)
	glog.V(50).Infoln("Looking for containers in", containers_path, "for entry", this.ContainerId)

	parent_node, err := zkc.Get(containers_path)
	if err != nil {
		return err
	}
	matches, err := parent_node.FilterChildrenRecursive(func(z *zk.Node) bool {
		host, _ := ParseHostPort(filepath.Base(z.GetPath()))
		// This is a filter function
		return host != this.ContainerId || !z.IsLeaf()
	})
	// Delete the matches
	for _, match := range matches {
		err = zkc.Delete(match.GetPath())
		if err != nil {
			glog.Warningln("Error de-registering", match.GetPath(), err)
			return err
		}
		glog.Infoln("De-registered", match.GetPath(), err)
	}

	// After the children have been removed, check the parent again
	parent_node, err = zkc.Get(containers_path)
	if err != nil {
		return err
	}
	if parent_node.Stats.NumChildren == 0 {
		glog.Infoln("No children under", containers_path, "removing parent.")
		err = zkc.Delete(containers_path)
		if err != nil {
			glog.Warningln("Error removing parent node", containers_path)
			return err
		} else {
			glog.Infoln("Removed parent node", containers_path)
		}
	}
	return nil
}
