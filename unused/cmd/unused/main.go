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
	fIgnore       string
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
	flag.StringVar(&fTags, "tags", "", "List of `build tags`")
	flag.StringVar(&fIgnore, "ignore", "", "Space separated list of checks to ignore, in the following format: 'import/path/file.go:Check1,Check2,...' Both the import path and file name sections support globbing, e.g. 'os/exec/*_test.go'")

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
	args := []string{
		"-tags", fTags,
		"-ignore", fIgnore,
	}
	args = append(args, flag.Args()...)
	lintutil.ProcessArgs("unused", l, args)
}
