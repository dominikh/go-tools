package main

import (
	"os"

	"github.com/cabify/go-tools/errcheck"
	"github.com/cabify/go-tools/lint/lintutil"
)

func main() {
	c := lintutil.CheckerConfig{
		Checker:     errcheck.NewChecker(),
		ExitNonZero: true,
	}
	lintutil.ProcessArgs("errcheck-ng", []lintutil.CheckerConfig{c}, os.Args[1:])
}
