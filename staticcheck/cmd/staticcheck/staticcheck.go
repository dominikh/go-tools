// staticcheck statically checks arguments to certain functions
package main // import "honnef.co/go/staticcheck/cmd/staticcheck"

import (
	"os"

	"honnef.co/go/lint"
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
	funcs := map[string]lint.Func{}
	for k, v := range staticcheck.Funcs {
		funcs[k] = v
	}
	if checkDubious {
		for k, v := range staticcheck.DubiousFuncs {
			funcs[k] = v
		}
	}
	lintutil.ProcessArgs("staticcheck", funcs, args)
}
