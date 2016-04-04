// staticcheck statically checks arguments to certain functions
package main // import "honnef.co/go/staticcheck/cmd/staticcheck"

import (
	"os"

	"honnef.co/go/lint/lintutil"
	"honnef.co/go/staticcheck"
)

func main() {
	lintutil.ProcessArgs("staticcheck", staticcheck.Funcs, os.Args[1:])
}
