package env

import (
	"flag"
)

func (this *Env) BindFlags() {
	flag.BoolVar(&this.ReadStdin, "stdin", false, "True to source env from standard input")
	flag.BoolVar(&this.Publish, "publish", false, "True to publish entries to destination path")
	flag.BoolVar(&this.Overwrite, "overwrite", false, "True to overwrite env value during publish")
}
