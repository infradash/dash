package registry

import (
	"fmt"
	"time"
)

type Timeout time.Duration

func (this *Timeout) UnmarshalJSON(s []byte) error {
	// unquote the string
	d, err := time.ParseDuration(string(s[1 : len(s)-1]))
	if err != nil {
		return err
	}
	*this = Timeout(d)
	return nil
}

func (this *Timeout) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", time.Duration(*this).String())), nil
}
