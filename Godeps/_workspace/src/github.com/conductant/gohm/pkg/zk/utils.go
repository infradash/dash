package zk

import (
	"github.com/golang/glog"
	"os"
	"strings"
)

// Determines the zookeeper hosts from ENV variables or
// use the default
func Hosts() []string {
	servers := strings.Split(DefaultZkHosts, ",")
	fromEnv := os.Getenv(EnvZkHosts)
	if len(fromEnv) > 0 {
		servers = strings.Split(fromEnv, ",")
	}
	glog.Infoln("zk-hosts:", servers)
	return servers
}

func append_string_slices(a, b []string) []string {
	l := len(a)
	ll := make([]string, l+len(b))
	copy(ll, a)
	for i, n := range b {
		ll[i+l] = n
	}
	return ll
}

func append_node_slices(a, b []*Node) []*Node {
	l := len(a)
	ll := make([]*Node, l+len(b))
	copy(ll, a)
	for i, n := range b {
		ll[i+l] = n
	}
	return ll
}
