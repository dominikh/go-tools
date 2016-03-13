package main

import (
	"flag"
	"fmt"
	"go/token"
	"go/types"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/kisielk/gotool"
	"golang.org/x/tools/go/loader"
)

var exitCode int

var (
	fConstants bool
	fFunctions bool
	fTypes     bool
	fVariables bool
)

func init() {
	flag.BoolVar(&fConstants, "c", true, "Report unused constants")
	flag.BoolVar(&fFunctions, "f", true, "Report unused functions and methods")
	flag.BoolVar(&fTypes, "t", true, "Report unused types")
	flag.BoolVar(&fVariables, "v", true, "Report unused variables")
}

func main() {
	flag.Parse()
	// FIXME check flag.NArgs
	paths := gotool.ImportPaths([]string{flag.Arg(0)})
	conf := loader.Config{AllowErrors: true}
	for _, path := range paths {
		conf.Import(path)
	}
	lprog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	defs := map[types.Object]bool{}
	var interfaces []*types.Interface
	for _, path := range paths {
		pkg := lprog.Package(path)
		if pkg == nil {
			log.Println("Couldn't load package", path)
			continue
		}
		for _, obj := range pkg.Defs {
			if obj == nil {
				continue
			}
			if obj, ok := obj.(*types.Var); ok {
				if typ, ok := obj.Type().(*types.Interface); ok {
					interfaces = append(interfaces, typ)
				}
			}
			if obj, ok := obj.(*types.TypeName); ok {
				if typ, ok := obj.Type().Underlying().(*types.Interface); ok {
					interfaces = append(interfaces, typ)
				}
			}
			if isVariable(obj) && !isPkgScope(obj) && !isField(obj) {
				// Skip variables that aren't package variables or struct fields
				continue
			}
			defs[obj] = false
		}
		for _, obj := range pkg.Uses {
			defs[obj] = true
		}
	}
	var reports Reports
	for obj, used := range defs {
		if obj.Pkg() == nil {
			continue
		}
		// TODO methods + reflection
		if !checkFlags(obj) {
			continue
		}
		if used {
			continue
		}
		if obj.Name() == "_" {
			continue
		}
		if obj.Exported() && (isPkgScope(obj) || isMethod(obj) || isField(obj)) {
			f := lprog.Fset.Position(obj.Pos()).Filename
			if !strings.HasSuffix(f, "_test.go") || strings.HasPrefix(obj.Name(), "Test") || strings.HasPrefix(obj.Name(), "Benchmark") {
				continue
			}
		}
		if isMain(obj) {
			continue
		}
		if isFunction(obj) && !isMethod(obj) && obj.Name() == "init" {
			continue
		}
		if isMethod(obj) && implements(obj, interfaces) {
			continue
		}
		reports = append(reports, Report{obj.Pos(), obj.Name()})
	}
	sort.Sort(reports)
	for _, report := range reports {
		fmt.Printf("%s: %s is unused\n", lprog.Fset.Position(report.pos), report.name)
	}

	os.Exit(exitCode)
}

func implements(obj types.Object, ifaces []*types.Interface) bool {
	recvType := obj.(*types.Func).Type().(*types.Signature).Recv().Type()
	for _, iface := range ifaces {
		if !types.Implements(recvType, iface) {
			continue
		}
		n := iface.NumMethods()
		for i := 0; i < n; i++ {
			if iface.Method(i).Name() == obj.Name() {
				return true
			}
		}
	}
	return false
}

func isPkgScope(obj types.Object) bool {
	return obj.Parent() == obj.Pkg().Scope()
}

func isMain(obj types.Object) bool {
	if obj.Pkg().Name() != "main" {
		return false
	}
	if obj.Name() != "main" {
		return false
	}
	if !isPkgScope(obj) {
		return false
	}
	if !isFunction(obj) {
		return false
	}
	if isMethod(obj) {
		return false
	}
	return true
}

func isFunction(obj types.Object) bool {
	_, ok := obj.(*types.Func)
	return ok
}

func isMethod(obj types.Object) bool {
	if !isFunction(obj) {
		return false
	}
	return obj.(*types.Func).Type().(*types.Signature).Recv() != nil
}

func isVariable(obj types.Object) bool {
	_, ok := obj.(*types.Var)
	return ok
}

func isConstant(obj types.Object) bool {
	_, ok := obj.(*types.Const)
	return ok
}

func isType(obj types.Object) bool {
	_, ok := obj.(*types.TypeName)
	return ok
}

func isField(obj types.Object) bool {
	if obj, ok := obj.(*types.Var); ok && obj.IsField() {
		return true
	}
	return false
}

func checkFlags(obj types.Object) bool {
	if isFunction(obj) && !fFunctions {
		return false
	}
	if isVariable(obj) && !fVariables {
		return false
	}
	if isConstant(obj) && !fConstants {
		return false
	}
	if isType(obj) && !fTypes {
		return false
	}
	return true
}

type Report struct {
	pos  token.Pos
	name string
}
type Reports []Report

func (l Reports) Len() int           { return len(l) }
func (l Reports) Less(i, j int) bool { return l[i].pos < l[j].pos }
func (l Reports) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
