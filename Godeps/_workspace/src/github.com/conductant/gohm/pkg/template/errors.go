package template

import (
	"errors"
	"fmt"
)

var (
	ErrMissingTemplateFunc = errors.New("no-template-func")
	ErrBadTemplateFunc     = errors.New("err-bad-template-func")
)

type NotSupported struct {
	Protocol string
}

func (this *NotSupported) Error() string {
	return fmt.Sprintf("not-supported: %s", this.Protocol)
}
