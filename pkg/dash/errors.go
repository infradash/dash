package dash

import (
	"errors"
)

var (
	ErrNotSupportedProtocol = errors.New("bad-url-protocol")
	ErrNoPath               = errors.New("no-path")
)
