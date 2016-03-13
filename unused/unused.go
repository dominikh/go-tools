package unused

import (
	"go/token"
	"go/types"
	"log"
	"strings"

	"golang.org/x/tools/go/loader"
)

// TODO correct name?
type CheckFlag int

const (
	CheckConstants CheckFlag = 1 << iota
	CheckFields
	CheckFunctions
	CheckTypes
	CheckVariables
)

type Checker struct {
	Flags CheckFlag
	Fset  *token.FileSet
}

func (c *Checker) checkConstants() bool { return (c.Flags & CheckConstants) > 0 }
func (c *Checker) checkFields() bool    { return (c.Flags & CheckFields) > 0 }
func (c *Checker) checkFunctions() bool { return (c.Flags & CheckFunctions) > 0 }
func (c *Checker) checkTypes() bool     { return (c.Flags & CheckTypes) > 0 }
func (c *Checker) checkVariables() bool { return (c.Flags & CheckVariables) > 0 }

func (c *Checker) Check(paths []string) ([]types.Object, error) {
	defs := map[types.Object]bool{}
	var interfaces []*types.Interface
	var unused []types.Object

	conf := loader.Config{}
	pkgs := map[string]bool{}
	for _, path := range paths {
		pkgs[path] = true
	}
	conf.TypeCheckFuncBodies = func(s string) bool {
		return pkgs[s]
	}
	for _, path := range paths {
		conf.ImportWithTests(path)
	}
	lprog, err := conf.Load()
	if err != nil {
		return nil, err
	}

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
			if _, ok := obj.(*types.PkgName); ok {
				continue
			}
			defs[obj] = false
		}
		for _, obj := range pkg.Uses {
			defs[obj] = true
		}
	}
	for obj, used := range defs {
		f := lprog.Fset.Position(obj.Pos()).Filename
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		if obj.Pkg() == nil {
			continue
		}
		// TODO methods + reflection
		if !c.checkFlags(obj) {
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
		unused = append(unused, obj)
	}
	c.Fset = lprog.Fset
	return unused, nil
}

func Check(paths []string, flags CheckFlag) ([]types.Object, error) {
	checker := Checker{Flags: flags}
	return checker.Check(paths)
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

func (c *Checker) checkFlags(obj types.Object) bool {
	if isFunction(obj) && !c.checkFunctions() {
		return false
	}
	if isVariable(obj) && !c.checkVariables() {
		return false
	}
	if isConstant(obj) && !c.checkConstants() {
		return false
	}
	if isType(obj) && !c.checkTypes() {
		return false
	}
	if isField(obj) && !c.checkFields() {
		return false
	}
	return true
}
