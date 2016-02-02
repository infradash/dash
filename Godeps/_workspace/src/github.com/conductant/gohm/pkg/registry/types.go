package registry

import (
	"io"
	"net/url"
)

type Version int32

const (
	InvalidVersion Version = -1
)

type Registry interface {
	io.Closer
	Id() url.URL
	Exists(Path) (bool, error)
	Get(Path) ([]byte, Version, error)
	Put(Path, []byte, bool) (Version, error)           // Create or set.
	PutVersion(Path, []byte, Version) (Version, error) // Create or set with CAS - not for ephemeral nodes
	Delete(Path) error
	DeleteVersion(Path, Version) error // Delete with CAS
	List(Path) ([]Path, error)
	Trigger(Trigger) (<-chan interface{}, chan<- int, error) // events channel, channel to stop, error
}
