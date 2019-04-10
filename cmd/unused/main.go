// unused reports unused identifiers (types, functions, ...) in your
// code.
package main // import "honnef.co/go/tools/cmd/unused"

import (
	"fmt"
	"log"
	"os"

	"honnef.co/go/tools/lint"
	"honnef.co/go/tools/lint/lintutil"
	"honnef.co/go/tools/unused"
)

var (
	fConstants    bool
	fFields       bool
	fFunctions    bool
	fTypes        bool
	fVariables    bool
	fDebug        string
	fWholeProgram bool
	fReflection   bool
)

func newChecker() *unused.Checker {
	checker := &unused.Checker{}
	if fDebug != "" {
		debug, err := os.Create(fDebug)
		if err != nil {
			log.Fatal("couldn't open debug file:", err)
		}
		checker.Debug = debug
	}

	checker.WholeProgram = fWholeProgram
	return checker
}

func main() {
	fmt.Fprintln(os.Stderr, "Unused has been deprecated. Please use staticcheck instead.")
	log.SetFlags(0)

	fs := lintutil.FlagSet("unused")
	fs.BoolVar(&fConstants, "consts", true, "Report unused constants")
	fs.BoolVar(&fFields, "fields", true, "Report unused fields")
	fs.BoolVar(&fFunctions, "funcs", true, "Report unused functions and methods")
	fs.BoolVar(&fTypes, "types", true, "Report unused types")
	fs.BoolVar(&fVariables, "vars", true, "Report unused variables")
	fs.StringVar(&fDebug, "debug", "", "Write a debug graph to `file`. Existing files will be overwritten.")
	fs.BoolVar(&fWholeProgram, "exported", false, "Treat arguments as a program and report unused exported identifiers")
	fs.BoolVar(&fReflection, "reflect", true, "Consider identifiers as used when it's likely they'll be accessed via reflection")
	fs.Parse(os.Args[1:])

	c := newChecker()
	lintutil.ProcessFlagSet([]lint.Checker{c}, fs)
}
