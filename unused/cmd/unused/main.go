package main

import (
	"flag"
	"fmt"
	"go/token"
	"log"
	"os"
	"sort"

	"honnef.co/go/unused"

	"github.com/kisielk/gotool"
)

var (
	fConstants bool
	fFields    bool
	fFunctions bool
	fTypes     bool
	fVariables bool
)

func init() {
	flag.BoolVar(&fConstants, "consts", true, "Report unused constants")
	flag.BoolVar(&fFields, "fields", false, "Report unused fields (may have false positives)")
	flag.BoolVar(&fFunctions, "funcs", true, "Report unused functions and methods")
	flag.BoolVar(&fTypes, "types", true, "Report unused types")
	flag.BoolVar(&fVariables, "vars", true, "Report unused variables")
}

func main() {
	flag.Parse()

	var flags unused.CheckFlag
	if fConstants {
		flags |= unused.CheckConstants
	}
	if fFields {
		flags |= unused.CheckFields
	}
	if fFunctions {
		flags |= unused.CheckFunctions
	}
	if fTypes {
		flags |= unused.CheckTypes
	}
	if fVariables {
		flags |= unused.CheckVariables
	}

	// FIXME check flag.NArgs
	paths := gotool.ImportPaths([]string{flag.Arg(0)})
	checker := unused.Checker{Flags: flags}
	objs, err := checker.Check(paths)
	if err != nil {
		log.Fatal(err)
	}
	var reports Reports
	for _, obj := range objs {
		reports = append(reports, Report{obj.Pos(), obj.Name()})
	}
	sort.Sort(reports)
	for _, report := range reports {
		fmt.Printf("%s: %s is unused\n", checker.Fset.Position(report.pos), report.name)
	}
	if len(reports) > 0 {
		os.Exit(1)
	}
}

type Report struct {
	pos  token.Pos
	name string
}
type Reports []Report

func (l Reports) Len() int           { return len(l) }
func (l Reports) Less(i, j int) bool { return l[i].pos < l[j].pos }
func (l Reports) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
