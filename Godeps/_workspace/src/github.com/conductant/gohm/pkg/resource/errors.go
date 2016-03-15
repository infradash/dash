package resource

import (
	"fmt"
)

type NotSupported struct {
	Protocol string
}

func (this *NotSupported) Error() string {
	return fmt.Sprintf("not-supported: %s", this.Protocol)
}
