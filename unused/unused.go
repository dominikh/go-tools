package unused // import "honnef.co/go/unused"

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/loader"
)

type CheckMode int

const (
	CheckConstants CheckMode = 1 << iota
	CheckFields
	CheckFunctions
	CheckTypes
	CheckVariables

	CheckAll = CheckConstants | CheckFields | CheckFunctions | CheckTypes | CheckVariables
)

type Unused struct {
	Obj      types.Object
	Position token.Position
}

type state struct {
	used  bool
	quiet bool
}

type Checker struct {
	Mode    CheckMode
	Verbose bool

	defs map[types.Object]*state
	pkg  *loader.PackageInfo
}

func NewChecker(mode CheckMode, verbose bool) *Checker {
	return &Checker{
		Mode:    mode,
		Verbose: verbose,
		defs:    make(map[types.Object]*state),
	}
}

func (c *Checker) checkConstants() bool { return (c.Mode & CheckConstants) > 0 }
func (c *Checker) checkFields() bool    { return (c.Mode & CheckFields) > 0 }
func (c *Checker) checkFunctions() bool { return (c.Mode & CheckFunctions) > 0 }
func (c *Checker) checkTypes() bool     { return (c.Mode & CheckTypes) > 0 }
func (c *Checker) checkVariables() bool { return (c.Mode & CheckVariables) > 0 }

func (c *Checker) markUsed(obj types.Object) {
	v, ok := c.defs[obj]
	if !ok {
		v = &state{}
		c.defs[obj] = v
	}
	v.used = true
}

func (c *Checker) markQuiet(obj types.Object) {
	v, ok := c.defs[obj]
	if !ok {
		v = &state{}
		c.defs[obj] = v
	}
	v.quiet = true
}

func (c *Checker) markCompositeLit(expr ast.Expr, typ types.Type) {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return
	}
	if isBasicStruct(lit.Elts) {
		c.markFields(typ)
	}
}

func (c *Checker) markFields(typ types.Type) {
	structType, ok := typ.Underlying().(*types.Struct)
	if !ok {
		return
	}
	n := structType.NumFields()
	for i := 0; i < n; i++ {
		field := structType.Field(i)
		c.markUsed(field)
	}
}

func (c *Checker) getType(expr ast.Expr) types.Type {
	switch t := expr.(type) {
	case *ast.StructType:
		// anonymous struct
		return c.pkg.TypeOf(t)
	case *ast.Ident:
		// named struct, slice, array or map
		typ, ok := c.pkg.ObjectOf(t).Type().(*types.Named)
		if !ok {
			return nil
		}
		return typ
	case *ast.ArrayType:
		return c.getType(t.Elt)
	}
	return nil
}

func (c *Checker) Visit(n ast.Node) ast.Visitor {
	node, ok := n.(*ast.CompositeLit)
	if !ok {
		return c
	}
	switch t := node.Type.(type) {
	case *ast.StructType, *ast.Ident, *ast.ArrayType:
		// struct{}{...}, T{...}, []T{...}, [...]T{...}
		typ1 := c.getType(t)
		if typ1 == nil {
			return c
		}
		switch typ2 := typ1.Underlying().(type) {
		case *types.Map:
			for _, elt := range node.Elts {
				c.markCompositeLit(elt.(*ast.KeyValueExpr).Key, typ2.Key())
				c.markCompositeLit(elt.(*ast.KeyValueExpr).Value, typ2.Elem())
			}
		case *types.Struct:
			if isBasicStruct(node.Elts) {
				c.markFields(typ1)
			}
		case *types.Slice, *types.Array:
			elemType := typ2.(interface {
				Elem() types.Type
			}).Elem()
			for _, elt := range node.Elts {
				if elt, ok := elt.(*ast.KeyValueExpr); ok {
					// S{1: {}}
					c.markCompositeLit(elt.Value, elemType)
					continue
				}

				// S{{}}
				c.markCompositeLit(elt, elemType)
			}
		}
	case *ast.MapType:
		keyType := c.getType(t.Key)
		valueType := c.getType(t.Value)
		for _, elt := range node.Elts {
			if keyType != nil {
				c.markCompositeLit(elt.(*ast.KeyValueExpr).Key, keyType)
			}
			if valueType != nil {
				c.markCompositeLit(elt.(*ast.KeyValueExpr).Value, valueType)
			}
		}
	}

	return c
}

func (c *Checker) Check(paths []string) ([]Unused, error) {
	// We resolve paths manually instead of relying on go/loader so
	// that our TypeCheckFuncBodies implementation continues to work.
	goFiles, err := resolveRelative(paths)
	if err != nil {
		return nil, err
	}
	var interfaces []*types.Interface
	var unused []Unused

	conf := loader.Config{}
	if !c.Verbose {
		conf.TypeChecker.Error = func(error) {}
	}
	pkgs := map[string]bool{}
	for _, path := range paths {
		pkgs[path] = true
		pkgs[path+"_test"] = true
	}
	if !goFiles {
		// Only type-check the packages we directly import. Unless
		// we're specifying a package in terms of individual files,
		// because then we don't know the import path.
		conf.TypeCheckFuncBodies = func(s string) bool {
			return pkgs[s]
		}
	}
	_, err = conf.FromArgs(paths, true)
	if err != nil {
		return nil, err
	}
	lprog, err := conf.Load()
	if err != nil {
		return nil, err
	}

	for _, c.pkg = range lprog.InitialPackages() {
		for _, obj := range c.pkg.Defs {
			if obj == nil {
				continue
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
			if _, ok := c.defs[obj]; !ok {
				c.defs[obj] = &state{}
			}
		}
		for _, tv := range c.pkg.Types {
			if typ, ok := tv.Type.(*types.Interface); ok {
				interfaces = append(interfaces, typ)
			}
		}
		for _, obj := range c.pkg.Uses {
			c.markUsed(obj)
		}
		for _, file := range c.pkg.Files {
			ast.Walk(c, file)
		}
	}
	for obj, state := range c.defs {
		if state.used {
			continue
		}
		if obj.Pkg() == nil {
			continue
		}
		if s, ok := obj.Type().Underlying().(*types.Struct); ok {
			n := s.NumFields()
			for i := 0; i < n; i++ {
				c.markQuiet(s.Field(i))
			}
		}
	}
	for obj, state := range c.defs {
		f := lprog.Fset.Position(obj.Pos()).Filename

		if obj.Pkg() == nil {
			continue
		}
		// TODO methods + reflection
		if !c.checkFlags(obj) {
			continue
		}
		if state.used || state.quiet {
			continue
		}

		if c.consideredUsed(obj, interfaces, f) {
			continue
		}

		unused = append(unused, Unused{
			Obj:      obj,
			Position: lprog.Fset.Position(obj.Pos()),
		})
	}
	return unused, nil
}

func isBasicStruct(elts []ast.Expr) bool {
	for _, elt := range elts {
		if _, ok := elt.(*ast.KeyValueExpr); !ok {
			return true
		}
	}
	return false
}

func resolveRelative(importPaths []string) (goFiles bool, err error) {
	if len(importPaths) == 0 {
		return false, nil
	}
	if strings.HasSuffix(importPaths[0], ".go") {
		// User is specifying a package in terms of .go files, don't resolve
		return true, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	for i, path := range importPaths {
		bpkg, err := build.Import(path, wd, build.FindOnly)
		if err != nil {
			return false, fmt.Errorf("can't load package %q: %v", path, err)
		}
		importPaths[i] = bpkg.ImportPath
	}
	return false, nil
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

func (c *Checker) consideredUsed(obj types.Object, interfaces []*types.Interface, f string) bool {
	// The blank identifier is used
	if obj.Name() == "_" {
		return true
	}

	// func main in package main is used
	if isMain(obj) {
		return true
	}

	// func init is used
	if isFunction(obj) && !isMethod(obj) && obj.Name() == "init" {
		return true
	}

	// methods that aid in implementing an interface are used
	if isMethod(obj) && implements(obj, interfaces) {
		return true
	}

	if obj.Exported() {
		// Exported methods and fields are always used
		if isMethod(obj) || isField(obj) {
			return true
		}

		// Test*, Benchmark* and Example* used, other exported identifiers are not
		if strings.HasSuffix(f, "_test.go") {
			return strings.HasPrefix(obj.Name(), "Test") ||
				strings.HasPrefix(obj.Name(), "Benchmark") ||
				strings.HasPrefix(obj.Name(), "Example")
		}

		// Package-level are used, except in package main
		if isPkgScope(obj) && c.pkg.Pkg.Name() != "main" {
			return true
		}
	}

	return false
}
