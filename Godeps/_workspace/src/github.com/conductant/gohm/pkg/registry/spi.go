package registry

import (
	"github.com/conductant/gohm/pkg/store"
	"golang.org/x/net/context"
	net "net/url"
)

// Registry backend implementations should follow this protocol to implement and register its services.
type Implementation func(ctx context.Context, url net.URL, dispose store.Dispose) (Registry, error)
type UrlSanitizer func(url net.URL) net.URL

func Register(protocol string, impl Implementation) {
	lock.Lock()
	defer lock.Unlock()
	protocols[scheme(protocol)] = impl
}

func RegisterSanitizer(protocol string, impl UrlSanitizer) {
	lock.Lock()
	defer lock.Unlock()
	sanitizers[scheme(protocol)] = impl
}
