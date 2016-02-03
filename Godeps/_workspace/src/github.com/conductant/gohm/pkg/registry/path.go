package registry

import (
	"fmt"
	p "path"
	"strings"
)

type Path interface {
	String() string
	Sub(...string) Path
	Base() string
	Dir() Path
	IsAbs() bool
	Parts() []string
}

type path string

func NewPath(s string, parts ...string) Path {
	return path(p.Clean(p.Join("/", s, p.Join(parts...))))
}

func NewPathf(format string, args ...interface{}) Path {
	return path(p.Clean(fmt.Sprintf(format, args...)))
}

func (this path) String() string {
	return string(this)
}

func (this path) IsAbs() bool {
	return p.IsAbs(string(this))
}
func (this path) Sub(parts ...string) Path {
	return path(p.Join(string(this), p.Join(parts...)))
}

func (this path) Base() string {
	return p.Base(string(this))
}

func (this path) Dir() Path {
	return path(p.Dir(string(this)))
}

func (this path) Parts() []string {
	return strings.Split(string(this), "/")
}
