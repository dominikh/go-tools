package main // import "honnef.co/go/unused/cmd/unused"

import (
	"flag"
	"fmt"
	"go/token"
	"go/types"
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
	fDebug     string
)

func init() {
	flag.BoolVar(&fConstants, "consts", true, "Report unused constants")
	flag.BoolVar(&fFields, "fields", true, "Report unused fields")
	flag.BoolVar(&fFunctions, "funcs", true, "Report unused functions and methods")
	flag.BoolVar(&fTypes, "types", true, "Report unused types")
	flag.BoolVar(&fVariables, "vars", true, "Report unused variables")
	flag.BoolVar(&fVerbose, "v", false, "Display type-checker errors")
	flag.StringVar(&fDebug, "debug", "", "Write a debug graph to `file`. Existing files will be overwritten.")

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

	if fDebug != "" {
		debug, err := os.Create(fDebug)
		if err != nil {
			log.Fatal("couldn't open debug file:", err)
		}
		checker.Debug = debug
	}

	unused, err := checker.Check(paths)
	if err != nil {
		log.Fatal(err)
	}
	var reports Reports
	for _, u := range unused {
		reports = append(reports, Report{u.Position, u.Obj.Name(), typString(u.Obj)})
	}
	sort.Sort(reports)
	for _, report := range reports {
		fmt.Printf("%s: %s %s is unused\n", report.pos, report.typ, report.name)
	}
	if len(reports) > 0 {
		os.Exit(1)
	}
}

func typString(obj types.Object) string {
	switch obj := obj.(type) {
	case *types.Func:
		return "func"
	case *types.Var:
		if obj.IsField() {
			return "field"
		}
		return "var"
	case *types.Const:
		return "const"
	case *types.TypeName:
		return "type"
	default:
		// log.Printf("%T", obj)
		return "identifier"
	}
}

type Report struct {
	pos  token.Position
	name string
	typ  string
}
type Reports []Report

func (l Reports) Len() int { return len(l) }
func (l Reports) Less(i, j int) bool {
	if l[i].pos.Filename < l[j].pos.Filename {
		return true
	} else if l[i].pos.Filename > l[j].pos.Filename {
		return false
	}
	if l[i].pos.Line < l[j].pos.Line {
		return true
	} else if l[i].pos.Line > l[j].pos.Line {
		return false
	}
	return l[i].pos.Column < l[j].pos.Column
}
func (l Reports) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
