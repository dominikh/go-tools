package main // import "honnef.co/go/unused/cmd/unused"

import (
	"flag"
	"fmt"
	"go/token"
	"go/types"
	"log"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"honnef.co/go/unused"

	"github.com/kisielk/gotool"
)

var (
	fConstants    bool
	fFields       bool
	fFunctions    bool
	fTypes        bool
	fVariables    bool
	fDebug        string
	fTags         string
	fWholeProgram bool
	fReflection   bool
)

func init() {
	flag.BoolVar(&fConstants, "consts", true, "Report unused constants")
	flag.BoolVar(&fFields, "fields", true, "Report unused fields")
	flag.BoolVar(&fFunctions, "funcs", true, "Report unused functions and methods")
	flag.BoolVar(&fTypes, "types", true, "Report unused types")
	flag.BoolVar(&fVariables, "vars", true, "Report unused variables")
	flag.StringVar(&fDebug, "debug", "", "Write a debug graph to `file`. Existing files will be overwritten.")
	flag.StringVar(&fTags, "tags", "", "List of build tags")
	flag.BoolVar(&fWholeProgram, "exported", false, "Treat arguments as a program and report unused exported identifiers")
	flag.BoolVar(&fReflection, "reflect", true, "Consider identifiers as used when it's likely they'll be accessed via reflection")

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

	checker.Tags = strings.Fields(fTags)
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

	paths := gotool.ImportPaths(flag.Args())
	checker := newChecker(mode)
	us, err := checker.Check(paths)
	if err == nil {
		printUnused(us)
		if len(us) > 0 {
			os.Exit(1)
		}
	}
	if err != nil && (len(paths) == 1 || fWholeProgram) {
		printErr(err, "")
		os.Exit(2)
	}

	anyUnused := false
	if err != nil && len(paths) > 1 {
		// Checking all packages at once potentially used a lot of
		// memory. While the Go runtime will gradually release it back
		// to the OS, we attempt to release it all in one go. While
		// this doesn't help with peak usage, it'll avoid concerned
		// user reports and it will reduce the average memory
		// consumption over time, which might matter if the tool runs
		// for a prolonged time.
		debug.FreeOSMemory()
		log.Println("Couldn't check all packages at once, will check each individually now")
		for _, path := range paths {
			checker = newChecker(mode)
			us, err := checker.Check([]string{path})
			if err != nil {
				printErr(err, path)
				continue
			}
			printUnused(us)
			if len(us) > 0 {
				anyUnused = true
			}
		}
	}

	if anyUnused {
		os.Exit(1)
	}
}

func printUnused(us []unused.Unused) {
	var reports Reports
	for _, u := range us {
		reports = append(reports, Report{u.Position, u.Obj.Name(), typString(u.Obj)})
	}
	sort.Sort(reports)
	for _, report := range reports {
		fmt.Printf("%s: %s %s is unused\n", report.pos, report.typ, report.name)
	}
}

func printErr(err error, path string) {
	if err2, ok := err.(unused.Error); ok {
		for pkg, errs := range err2.Errors {
			max := 4
			if max > len(errs) {
				max = len(errs)
			}
			log.Println("#", pkg)
			for _, err := range errs[:max] {
				log.Println(err)
			}
			if max < len(errs) {
				log.Println("too many errors")
			}
		}
	} else {
		if path != "" {
			log.Println("#", path)
		}
		log.Println(err)
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
