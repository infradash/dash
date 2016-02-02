package registry

// Special marker interface implemented only by Create, Change, Delete, and Members
type kind int

const (
	kindCreate kind = iota
	kindChange
	kindDelete
	kindMembers
)

// The Trigger interface is designed so that it's not possible to implement
// this interface outside this package.
type Trigger interface {
	kind() kind
}

func (this Create) kind() kind {
	return kindCreate
}

func (this Change) kind() kind {
	return kindChange
}

func (this Delete) kind() kind {
	return kindDelete
}

func (this Members) kind() kind {
	return kindMembers
}

type Create struct {
	Path    `json:"path"`
	Trigger `json:"-"`
}

type Change struct {
	Path    `json:"path"`
	Trigger `json:"-"`
}

type Delete struct {
	Path    `json:"path"`
	Trigger `json:"-"`
}

// For equality, set both min and max.  For not equals, set min, max and OutsideRange to true.
type Members struct {
	Path         `json:"path"`
	Min          *int `json:"min,omitempty"`
	Max          *int `json:"max,omitempty"`
	Delta        *int `json:"delta,omitempty"`         // delta of count
	OutsideRange bool `json:"outside_range,omitempty"` // default is within range.  true for outside range.
	Trigger      `json:"-"`
}

func (this *Members) SetMin(min int) *Members {
	this.Min = &min
	return this
}

func (this *Members) SetMax(max int) *Members {
	this.Max = &max
	return this
}

func (this *Members) SetDelta(d int) *Members {
	this.Delta = &d
	return this
}

func (this *Members) SetOutsideRange(b bool) *Members {
	this.OutsideRange = b
	return this
}
