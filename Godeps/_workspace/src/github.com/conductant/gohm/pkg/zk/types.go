package zk

import (
	"github.com/samuel/go-zookeeper/zk"
	"io"
	"time"
)

const (
	StateUnknown           = zk.StateUnknown
	StateDisconnected      = zk.StateDisconnected
	StateConnecting        = zk.StateConnecting
	StateAuthFailed        = zk.StateAuthFailed
	StateConnectedReadOnly = zk.StateConnectedReadOnly
	StateSaslAuthenticated = zk.StateSaslAuthenticated
	StateExpired           = zk.StateExpired
	StateConnected         = zk.StateConnected
	StateHasSession        = zk.StateHasSession

	EventNodeCreated         = zk.EventNodeCreated
	EventNodeDataChanged     = zk.EventNodeDataChanged
	EventNodeDeleted         = zk.EventNodeDeleted
	EventNodeChildrenChanged = zk.EventNodeChildrenChanged

	DefaultTimeout = 1 * time.Hour
)

type Node struct {
	Path    string
	Value   []byte
	Members []string
	Stats   *zk.Stat
	Leaf    bool
	client  *client
}

type Event struct {
	zk.Event
	Action string
	Note   string
}

type Service interface {
	io.Closer

	Reconnect() error
	Events() <-chan Event
	CreateNode(string, []byte, bool) (*Node, error)
	PutNode(string, []byte, bool) (*Node, error)
	GetNode(string) (*Node, error)
	DeleteNode(string) error
	WatchOnce(string, func(Event)) (chan<- bool, error)
	WatchOnceChildren(string, func(Event)) (chan<- bool, error)
	Watch(string, func(Event) bool, ...func(error)) (chan<- bool, error)
	WatchChildren(string, func(Event) bool, ...func(error)) (chan<- bool, error)
}
