package registry

import (
	"golang.org/x/net/context"
	net "net/url"
)

// Generic implementation using the registry API to implement template package's Source
// function. This function is registered by specific backend implementation packages (e.g. zk).
func Source(ctx context.Context, url string) ([]byte, error) {
	u, err := net.Parse(url)
	if err != nil {
		return nil, err
	}
	reg, err := Dial(ctx, u.String())
	if err != nil {
		return nil, err
	}
	bytes, _, err := reg.Get(NewPath(u.Path))
	return bytes, err
}
