package dash

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/registry"
	"github.com/qorio/maestro/pkg/template"
	"github.com/qorio/maestro/pkg/zk"
	"io"
	"sort"
	"strings"
)

func (this *EnvSource) IsZero() bool {
	if this.RegistryEntryBase.CheckRequires() {
		return false
	}
	return this.Url == ""
}

func (this *EnvSource) Source(authToken string, zc zk.ZK) func() ([]string, map[string]interface{}) {
	switch {
	case strings.Index(this.Url, "http") == 0:
		return this.EnvFromUrl(this.Url, authToken, zc)
	case strings.Index(this.Url, "zk://") == 0:
		return this.EnvFromZkPath(this.Url[len("zk://"):], zc)
	default:
		return this.EnvFromZk(zc)
	}
}

func (this *EnvSource) EnvFromUrl(url string, authToken string, zc zk.ZK) func() ([]string, map[string]interface{}) {
	return func() ([]string, map[string]interface{}) {
		body, _, err := template.FetchUrl(url, map[string]string{"Authorization": "Bearer " + authToken}, zc)
		if err != nil {
			return []string{}, map[string]interface{}{}
		}
		return parse_env(bytes.NewBufferString(body))
	}
}

func (this *EnvSource) EnvFromReader(reader io.Reader) func() ([]string, map[string]interface{}) {
	return func() ([]string, map[string]interface{}) {
		return parse_env(reader)
	}
}

func (this *EnvSource) EnvFromZk(zc zk.ZK) func() ([]string, map[string]interface{}) {
	// Path takes precedence over derived path based on domain, service, version, etc.
	var env_path string
	if this.Path != "" {
		env_path = this.Path
	} else {
		key, _, err := RegistryKeyValue(KEnvRoot, this)
		if err != nil {
			panic(err)
		}
		env_path = key
	}
	return this.EnvFromZkPath(env_path, zc)
}

func (this *EnvSource) EnvFromZkPath(env_path string, zc zk.ZK) func() ([]string, map[string]interface{}) {
	return func() ([]string, map[string]interface{}) {

		glog.Infoln("Loading env from", env_path)
		root_node, err := zk.Follow(zc, registry.Path(env_path))
		switch err {
		case nil:
		case zk.ErrNotExist:
			return []string{}, map[string]interface{}{}
		default:
			panic(err)
		}

		// Just get the entire set of values and export them as environment variables
		all, err := root_node.FilterChildrenRecursive(func(z *zk.Node) bool {
			return !z.IsLeaf() // filter out parent nodes
		})

		if err != nil {
			panic(err)
		}

		keys := make([]string, 0)
		env := make(map[string]interface{})
		for _, node := range all {
			key, value, err := zk.Resolve(zc, registry.Path(node.GetBasename()), node.GetValueString())
			if err != nil {
				panic(errors.New("bad env reference:" + key.Path() + "=>" + value))
			}

			env[key.Path()] = value
			keys = append(keys, key.Path())
		}
		sort.Strings(keys)
		return keys, env
	}
}

func parse_env(reader io.Reader) ([]string, map[string]interface{}) {
	keys := make([]string, 0)
	env := make(map[string]interface{})
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		i := strings.Index(line, "=")
		if i > 0 {
			key := line[0:i]
			value := line[i+1:]
			keys = append(keys, key)
			env[key] = value
		}
	}
	sort.Strings(keys)
	return keys, env
}
