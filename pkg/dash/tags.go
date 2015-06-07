package dash

import (
	"github.com/deckarep/golang-set"
)

func (this *QualifyByTags) Matches(other []string) bool {
	a := mapset.NewSet()
	b := mapset.NewSet()
	for _, v := range this.Tags {
		a.Add(v)
	}
	for _, v := range other {
		b.Add(v)
	}
	return a.Intersect(b).Cardinality() > 0
}
