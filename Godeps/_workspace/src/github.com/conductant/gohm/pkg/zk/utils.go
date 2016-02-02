package zk

import (
	"github.com/golang/glog"
	"os"
	"strings"
)

func Hosts() []string {
	servers := []string{"localhost:2181"}
	list := os.Getenv("ZK_HOSTS")
	if len(list) > 0 {
		servers = strings.Split(list, ",")
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
