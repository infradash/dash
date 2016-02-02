package registry

import (
	"fmt"
)

type NotSupportedProtocol struct {
	Protocol string
}

func (this *NotSupportedProtocol) Error() string {
	return fmt.Sprintf("not-supported: %s", this.Protocol)
}
