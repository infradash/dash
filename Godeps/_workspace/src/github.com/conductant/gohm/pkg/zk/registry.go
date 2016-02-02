package zk

import (
	. "github.com/conductant/gohm/pkg/registry"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"net/url"
	"strings"
)

func init() {
	Register("zk", NewService)
}

// Optional parameter is timeout, in Duration.
func NewService(ctx context.Context, url url.URL) (Registry, error) {
	// Look for a duration and use that as the timeout
	timeout := ContextGetTimeout(ctx)
	servers := strings.Split(url.Host, ",") // host:port,host:port,...
	return Connect(servers, timeout)
}

func (this *client) Id() url.URL {
	return this.url
}

func (this *client) Exists(key Path) (bool, error) {
	_, err := this.GetNode(key.String())
	switch err {
	case ErrNotExist:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
}

func (this *client) Get(key Path) ([]byte, Version, error) {
	n, err := this.GetNode(key.String())
	if err != nil {
		return nil, InvalidVersion, err
	}
	return n.Value, Version(n.Version()), nil
}

func (this *client) List(key Path) ([]Path, error) {
	n, err := this.GetNode(key.String())
	if err != nil {
		return nil, err
	}
	children, err := n.Children()
	if err != nil {
		return nil, err
	}
	paths := []Path{}
	for _, n := range children {
		paths = append(paths, NewPath(n.Path))
	}
	return paths, nil
}

func (this *client) Delete(key Path) error {
	return this.DeleteNode(key.String())
}

func (this *client) DeleteVersion(key Path, version Version) error {
	return this.conn.Delete(key.String(), int32(version))
}

func (this *client) Put(key Path, value []byte, ephemeral bool) (Version, error) {
	n, err := this.PutNode(key.String(), value, ephemeral)
	if err != nil {
		return InvalidVersion, err
	}
	return Version(n.Version()), nil
}

func (this *client) PutVersion(key Path, value []byte, version Version) (Version, error) {
	stat, err := this.conn.Set(key.String(), value, int32(version))
	if err != nil {
		return InvalidVersion, err
	} else {
		return Version(stat.Version), nil
	}
}

func (this *client) Trigger(t Trigger) (<-chan interface{}, chan<- int, error) {
	stop := make(chan int)
	events := make(chan interface{}, 8)

	var cStop chan<- int
	var cStopped <-chan error
	var err error
	switch t := t.(type) {
	case Create:
		cStop, cStopped, err = this.Watch(t.Path.String(),
			func(e Event) {
				if e.Type == EventNodeCreated {
					events <- e
				}
			})
		if err != nil {
			return nil, nil, err
		}
	case Change:
		cStop, cStopped, err = this.Watch(t.Path.String(),
			func(e Event) {
				if e.Type == EventNodeDataChanged {
					events <- e
				}
			})
		if err != nil {
			return nil, nil, err
		}
	case Delete:
		cStop, cStopped, err = this.Watch(t.Path.String(),
			func(e Event) {
				if e.Type == EventNodeDeleted {
					events <- e
				}
			})
		if err != nil {
			return nil, nil, err
		}
	case Members:
		// TODO - Implement the matching criteria using min/max/delta, etc.
		cStop, cStopped, err = this.WatchChildren(t.Path.String(),
			func(e Event) {
				if e.Type == EventNodeChildrenChanged {
					events <- e
				}
			})
		if err != nil {
			return nil, nil, err
		}
	}
	go func() {
		// Stop the watch
		c := <-stop
		cStop <- c
		glog.Infoln("Waiting for user callbacks to finish")
		<-cStopped
		glog.Infoln("Stopped.")
	}()
	return events, stop, nil
}
