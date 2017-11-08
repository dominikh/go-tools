package main // import "honnef.co/go/tools/cmd/errcheck-ng"

import (
	"os"

	"honnef.co/go/tools/errcheck"
	"honnef.co/go/tools/lint/lintutil"
)

func main() {
	c := lintutil.CheckerConfig{
		Checker:     errcheck.NewChecker(),
		ExitNonZero: true,
	}
	lintutil.ProcessArgs("errcheck-ng", []lintutil.CheckerConfig{c}, os.Args[1:])
}
