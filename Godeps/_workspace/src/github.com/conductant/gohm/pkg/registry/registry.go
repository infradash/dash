package registry

import (
	"fmt"
	"github.com/conductant/gohm/pkg/store"
	"golang.org/x/net/context"
	net "net/url"
	"strings"
	"sync"
)

type scheme string

var (
	regStore = store.NewRefCountStore(keyFromRegistry).Start()

	lock       sync.Mutex
	protocols  = map[scheme]Implementation{}
	sanitizers = map[scheme]UrlSanitizer{}
)

// Get an instance of the registry.  The url can specify host(s) such as
// zk://host1:2181,host2:2181,host3:2181/other/parts/of/path
// The protocol / scheme portion is used to dispatch to different registry implementations (e.g. zk: for zookeeper
// etcd: for ectd, etc.)
func Dial(ctx context.Context, url string) (Registry, error) {
	u, err := net.Parse(url)
	if err != nil {
		return nil, err
	}
	// This allows the implementation to deal with cases where Host is not set, etc., so the implementations
	// have an opportunity to use its own environment variables, etc. to fix up the url.
	if sanitizer, has := sanitizers[scheme(u.Scheme)]; has {
		clean := sanitizer(*u)
		u = &clean
	}
	impl, has := protocols[scheme(u.Scheme)]
	if !has {
		return nil, &NotSupportedProtocol{u.Scheme}
	}
	reg, err := regStore.Get(keyFromUrl(u), func(d store.Dispose) (store.Key, store.Object, error) {
		obj, err := impl(ctx, *u, d)
		if err != nil {
			return nil, nil, err
		}
		key := keyFromRegistry(obj)
		return key, store.Object(obj), nil
	})
	if reg != nil {
		return reg.(Registry), err
	} else {
		return nil, err
	}
}

func keyFromUrl(u *net.URL) store.Key {
	return store.Key(fmt.Sprintf("%s://%s", u.Scheme, u.Host))
}

// Function that returns the store key given a registry.
func keyFromRegistry(obj store.Object) store.Key {
	switch obj := obj.(type) {
	case Registry:
		u := obj.Id()
		return keyFromUrl(&u)
	default:
		panic(fmt.Errorf("Not a registry object %v", obj))
	}
}

// Given the fully specified url that includes protocol and host and path,
// traverses symlinks where the value of a node is a pointer url to another registry node
// It's possible that the pointer points to a different registry.
// The returned url includes protocol and host information
func FollowUrl(ctx context.Context, url net.URL) (net.URL, []byte, Version, error) {
	reg, err := Dial(ctx, url.String())
	if reg != nil {
		defer reg.Close()
	}
	if err != nil {
		return url, nil, InvalidVersion, err
	}
	return follow(ctx, reg, NewPath(url.Path))
}

// Traverses symlinks where the value of a node is a pointer url to another registry node
// It's possible that the pointer points to a different registry.
func follow(ctx context.Context, reg Registry, path Path) (net.URL, []byte, Version, error) {
	here := reg.Id()
	here.Path = path.String()
	if len(path.String()) == 0 {
		return here, nil, InvalidVersion, nil
	}
	v, ver, err := reg.Get(path)
	if err != nil {
		return here, nil, InvalidVersion, err
	}
	s := string(v)
	if strings.Contains(s, "://") {
		url, err := net.Parse(s)
		if err != nil {
			return here, nil, InvalidVersion, err
		}
		next, err := Dial(ctx, s)
		if next != nil {
			defer next.Close()
		}
		if err != nil {
			return here, nil, InvalidVersion, err
		}
		return follow(ctx, next, NewPath(url.Path))
	} else {
		return here, v, ver, nil
	}
}
