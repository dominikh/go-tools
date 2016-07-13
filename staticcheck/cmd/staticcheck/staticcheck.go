// staticcheck statically checks arguments to certain functions
package main // import "honnef.co/go/staticcheck/cmd/staticcheck"

import (
	"os"

	"honnef.co/go/lint/lintutil"
	"honnef.co/go/staticcheck"
)

var checkDubious bool

func main() {
	var args []string
	for _, arg := range os.Args[1:] {
		if arg == "-dubious" {
			checkDubious = true
			continue
		}
		args = append(args, arg)
	}
	lintutil.ProcessArgs("staticcheck", staticcheck.Funcs, args)
	if checkDubious {
		lintutil.ProcessArgs("staticcheck", staticcheck.DubiousFuncs, args)
	}
}
