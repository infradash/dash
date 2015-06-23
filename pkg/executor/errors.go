package executor

import (
	"errors"
)

var (
	ErrBadTemplate = errors.New("bad-template")
	ErrBadPath     = errors.New("bad-path")
)
