package restart

import (
	"errors"
)

var (
	ErrBadUrl    = errors.New("bad-url")
	ErrBadConfig = errors.New("bad-config")
	ErrNotLoaded = errors.New("not-loaded")
	ErrNoConfig  = errors.New("no-config")
)
