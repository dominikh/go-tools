package main // import "honnef.co/go/unused/cmd/unused"

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
	fVerbose   bool
)

func init() {
	flag.BoolVar(&fConstants, "consts", true, "Report unused constants")
	flag.BoolVar(&fFields, "fields", false, "Report unused fields")
	flag.BoolVar(&fFunctions, "funcs", true, "Report unused functions and methods")
	flag.BoolVar(&fTypes, "types", true, "Report unused types")
	flag.BoolVar(&fVariables, "vars", true, "Report unused variables")
	flag.BoolVar(&fVerbose, "v", false, "Display type-checker errors")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [packages]\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
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

	paths := gotool.ImportPaths(flag.Args())
	checker := unused.NewChecker(mode, fVerbose)
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
