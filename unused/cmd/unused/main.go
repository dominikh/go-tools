package main // import "honnef.co/go/unused/cmd/unused"

import (
	"flag"
	"fmt"
	"log"
	"os"

	"honnef.co/go/lint/lintutil"
	"honnef.co/go/unused"
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
	fTags         string
)

func init() {
	flag.BoolVar(&fConstants, "consts", true, "Report unused constants")
	flag.BoolVar(&fFields, "fields", true, "Report unused fields")
	flag.BoolVar(&fFunctions, "funcs", true, "Report unused functions and methods")
	flag.BoolVar(&fTypes, "types", true, "Report unused types")
	flag.BoolVar(&fVariables, "vars", true, "Report unused variables")
	flag.StringVar(&fDebug, "debug", "", "Write a debug graph to `file`. Existing files will be overwritten.")
	flag.BoolVar(&fWholeProgram, "exported", false, "Treat arguments as a program and report unused exported identifiers")
	flag.BoolVar(&fReflection, "reflect", true, "Consider identifiers as used when it's likely they'll be accessed via reflection")
	flag.StringVar(&fTags, "tags", "", "Build tags")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [packages]\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func newChecker(mode unused.CheckMode) *unused.Checker {
	checker := unused.NewChecker(mode)

	if fDebug != "" {
		debug, err := os.Create(fDebug)
		if err != nil {
			log.Fatal("couldn't open debug file:", err)
		}
		checker.Debug = debug
	}

	checker.WholeProgram = fWholeProgram
	checker.ConsiderReflection = fReflection
	return checker
}

func main() {
	log.SetFlags(0)
	flag.Parse()
	var mode unused.CheckMode
	if fConstants {
		mode |= unused.CheckConstants
	}
	if fFields {
		mode |= unused.CheckFields
	}
	if fFunctions {
		mode |= unused.CheckFunctions
	}
	if fTypes {
		mode |= unused.CheckTypes
	}
	if fVariables {
		mode |= unused.CheckVariables
	}

	checker := newChecker(mode)
	l := unused.NewLintChecker(checker)
	args := flag.Args()
	if len(fTags) > 0 {
		args = append([]string{"-tags", fTags}, args...)
	}
	lintutil.ProcessArgs("unused", l, args)
}
