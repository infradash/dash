package dash

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/zk"
	"os"
	"sort"
	"strings"
)

type Env struct {
	ZkSettings
	EnvSource // specifies input

	RegistryEntryBase // specifies output

	Publish   bool
	Overwrite bool

	zk zk.ZK
}

func (this *Env) EnvFromStdin() func() ([]string, map[string]string) {
	return func() ([]string, map[string]string) {

		glog.Infoln("Loading env from stdin")

		keys := make([]string, 0)
		env := make(map[string]string)
		scanner := bufio.NewScanner(os.Stdin)
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
		glog.Infoln("Loaded", len(keys), "entries from stdin")
		return keys, env
	}
}

func (this *Env) EnvFromZk() func() ([]string, map[string]string) {
	return func() ([]string, map[string]string) {

		path := this.Path
		glog.Infoln("Loading env from", path)
		if path == "" {
			panic(errors.New("no-path-env-param-not-set"))
		}

		root_node, err := this.zk.Get(path)
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
			key := node.GetBasename()
			value := node.GetValueString()
			env[key] = value
			keys = append(keys, key)
		}
		sort.Strings(keys)
		glog.Infoln("Loaded", len(keys), "entries from", path)
		return keys, env
	}
}

func (this *Env) Run() error {

	var source func() ([]string, map[string]string) = nil

	if this.ReadStdin {
		source = this.EnvFromStdin()
	} else {

		// Simple check to see if path and destination env path are the same
		check, _, _ := RegistryKeyValue(KEnvRoot, this)
		if this.Path == check {
			glog.Infoln("Source", this.Path, "and destination", check, "are the same. Nothing to do.")
			return nil
		}

		zookeeper, err := zk.Connect(strings.Split(this.Hosts, ","), this.Timeout)
		if err != nil {
			panic(err)
		}
		this.zk = zookeeper

		source = this.EnvFromZk()
	}

	if !this.RegistryEntryBase.CheckRequires() {
		return errors.New("no-path")
	}

	// Now run it
	vars, env := source()

	if this.Publish && this.zk == nil {
		zookeeper, err := zk.Connect(strings.Split(this.Hosts, ","), this.Timeout)
		if err != nil {
			panic(err)
		}
		this.zk = zookeeper
	}

	for _, k := range vars {

		entry := &RegistryEnvEntry{RegistryEntryBase: this.RegistryEntryBase, EnvName: k, EnvValue: env[k]}
		key, value, err := RegistryKeyValue(KEnv, entry)

		if err != nil {
			return err
		}

		if this.Publish {
			// Upsert
			n, err := this.zk.Get(key)
			switch {

			case err == zk.ErrNotExist:
				n, err = this.zk.Create(key, []byte(value))
				glog.Infoln("Created", key, "err=", err)

			case err != nil:
				glog.Warningln("Error upsert", key, "err=", err)
				continue

			}

			if value != n.GetValueString() {

				if this.Overwrite {
					if err := n.Set([]byte(value)); err != nil {
						glog.Warningln("Error upsert", key, "err=", err)
					} else {
						glog.Warningln("Committed", key, "err=", err)
					}
				} else {
					glog.Infoln("No overwrite -- key=", key, "source=", value, "to=", n.GetValueString())
				}

			} else {
				glog.Infoln("Key", key, "have same values. Nothing updated.")
			}

		} else {
			fmt.Fprintf(os.Stdout, "%s/%s=>%s\n", this.Path, k, key)
		}
	}

	if this.zk != nil {
		this.zk.Close()
	}
	return nil
}
