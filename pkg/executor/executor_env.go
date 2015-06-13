package executor

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/registry"
	"github.com/qorio/maestro/pkg/zk"
	"sort"
	"strings"
	"text/template"
)

func (this *Executor) EnvFromStdin() func() ([]string, map[string]string) {
	return func() ([]string, map[string]string) {
		keys := make([]string, 0)
		env := make(map[string]string)
		scanner := bufio.NewScanner(this.Stdin())
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
}

func (this *Executor) EnvFromZk() func() ([]string, map[string]string) {
	return func() ([]string, map[string]string) {

		if !this.EnvSource.RegistryEntryBase.CheckRequires() {
			panic(errors.New("no-path"))
		}

		var env_path string
		if this.Path != "" {
			env_path = this.Path
		} else {
			key, _, err := RegistryKeyValue(KEnvRoot, this.EnvSource.RegistryEntryBase)
			if err != nil {
				panic(err)
			}
			env_path = key
		}

		glog.Infoln("Loading env from", env_path)

		root_node, err := this.zk.Get(env_path)
		if err != nil {
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
		env := make(map[string]string)
		for _, node := range all {
			key, value, err := zk.Resolve(this.zk, registry.Path(node.GetBasename()), node.GetValueString())
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

func (this *Executor) ParseCustomVars() error {
	this.customVars = make(map[string]*template.Template)

	for _, expression := range strings.Split(this.CustomVarsCommaSeparated, ",") {
		parts := strings.Split(expression, "=")
		if len(parts) != 2 {
			return errors.New("invalid-template:" + expression)
		}
		key, exp := parts[0], parts[1]
		if t, err := template.New(key).Parse(exp); err != nil {
			return err
		} else {
			this.customVars[key] = t
		}
	}
	return nil
}

func (this *Executor) injectCustomVars(env map[string]string) ([]string, error) {
	for k, t := range this.customVars {
		var buff bytes.Buffer
		if err := t.Execute(&buff, this); err != nil {
			return nil, err
		} else {
			env[k] = buff.String()
			glog.Infoln("CustomVar:", k, buff.String())
		}
	}

	keys := make([]string, 0)
	for k, _ := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// Given the environments apply any substitutions to the command and args
// the format is same as template expressions e.g. {{.RUN_BINARY}}
// This allows the environment to be passed to the command even if the child process
// does not look at environment variables.
func (this *Executor) applyCmdSubstitutions(env map[string]string) error {
	templates := make([]*template.Template, len(this.Args)+1)
	templates[0] = template.Must(template.New(this.Cmd).Parse(this.Cmd))
	for i, a := range this.Args {
		templates[i+1] = template.Must(template.New(a).Parse(a))
	}

	// Now apply values
	applied := make([]string, len(templates))
	for i, t := range templates {
		var buff bytes.Buffer
		if err := t.Execute(&buff, env); err != nil {
			return err
		} else {
			applied[i] = buff.String()
		}
	}

	this.ExecCmd = applied[0]
	this.ExecArgs = applied[1:]
	glog.Infoln("ExecCmd=", this.ExecCmd, "ExecArgs=", this.ExecArgs)
	return nil
}
