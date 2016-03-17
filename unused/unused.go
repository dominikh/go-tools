package unused // import "honnef.co/go/unused"

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types/typeutil"
)

type graph struct {
	roots []*graphNode
	nodes map[interface{}]*graphNode
}

func (g *graph) markUsedBy(obj, usedBy interface{}) {
	if obj == usedBy {
		return
	}
	objNode := g.getNode(obj)
	usedByNode := g.getNode(usedBy)
	usedByNode.uses[objNode] = struct{}{}
}

func (g *graph) getNode(obj interface{}) *graphNode {
	node, ok := g.nodes[obj]
	if !ok {
		node = &graphNode{obj: obj, uses: make(map[*graphNode]struct{})}
		g.nodes[obj] = node
	}
	return node
}

func (g *graph) markScopeUsed(scope *types.Scope) {
	g.markScopeUsedBy(scope, nil)
}

func (g *graph) markScopeUsedBy(s1, s2 *types.Scope) {
	if s2 != nil {
		g.markUsedBy(s1, s2)
	}
	n := s1.NumChildren()
	for i := 0; i < n; i++ {
		g.markScopeUsedBy(s1.Child(i), s1)
	}
}

type graphNode struct {
	obj   interface{}
	uses  map[*graphNode]struct{}
	used  bool
	quiet bool
}

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

	graph *graph

	defs       map[types.Object]*state
	interfaces []*types.Interface
	structs    []*types.Named
	pkg        *loader.PackageInfo
	msCache    typeutil.MethodSetCache
	lprog      *loader.Program
}

func NewChecker(mode CheckMode, verbose bool) *Checker {
	return &Checker{
		Mode:    mode,
		Verbose: verbose,
		defs:    make(map[types.Object]*state),
		graph: &graph{
			nodes: make(map[interface{}]*graphNode),
		},
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

func (c *Checker) Visit(node ast.Node) ast.Visitor {
	return nil
}

func (c *Checker) Check(paths []string) ([]Unused, error) {
	// We resolve paths manually instead of relying on go/loader so
	// that our TypeCheckFuncBodies implementation continues to work.
	goFiles, err := resolveRelative(paths)
	if err != nil {
		return nil, err
	}
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
	c.lprog, err = conf.Load()
	if err != nil {
		return nil, err
	}

	for _, c.pkg = range c.lprog.InitialPackages() {
		for _, obj := range c.pkg.Defs {
			if obj == nil {
				continue
			}
			// if _, ok := obj.(*types.PkgName); ok {
			// 	continue
			// }

			if obj, ok := obj.(*types.TypeName); ok {
				c.graph.markUsedBy(obj.Type().Underlying(), obj.Type())
				c.graph.markUsedBy(obj.Type(), obj) // TODO is this needed?
				c.graph.markUsedBy(obj, obj.Type())
			}

			// FIXME(dominikh): we don't really want _ as roots. A _
			// variable in an otherwise unused function shouldn't mark
			// anything as used. However, _ doesn't seem to have a
			// scope associated with it.
			if obj, ok := obj.(*types.Var); ok {
				if obj.Name() == "_" {
					emptyNode := c.graph.getNode(obj)
					c.graph.roots = append(c.graph.roots, emptyNode)
				} else {
					if obj.Parent() != obj.Pkg().Scope() {
						c.graph.markUsedBy(obj, obj.Parent())
					}
				}
			}

			if fn, ok := obj.(*types.Func); ok {
				c.graph.markUsedBy(fn, fn.Type())
			}

			if obj, ok := obj.(interface {
				Scope() *types.Scope
			}); ok {
				scope := obj.Scope()
				c.graph.markUsedBy(scope, obj)
				c.graph.markScopeUsed(scope)
			}

			if c.isRoot(obj, false) {
				node := c.graph.getNode(obj)
				c.graph.roots = append(c.graph.roots, node)
				if obj, ok := obj.(*types.PkgName); ok {
					scope := obj.Pkg().Scope()
					c.graph.markUsedBy(scope, obj)
				}
			}
		}

		for ident, usedObj := range c.pkg.Uses {
			if _, ok := usedObj.(*types.PkgName); ok {
				continue
			}
			pos := ident.Pos()
			scope := c.pkg.Pkg.Scope().Innermost(pos)
			c.graph.markUsedBy(usedObj, scope)

			if obj, ok := usedObj.(*types.Var); ok {
				c.graph.markUsedBy(obj.Type(), obj)
			}
		}

		for _, tv := range c.pkg.Types {
			if iface, ok := tv.Type.(*types.Interface); ok {
				if iface.NumMethods() == 0 {
					continue
				}
				for _, node := range c.graph.nodes {
					obj, ok := node.obj.(types.Object)
					if !ok {
						continue
					}
					// TODO check pointer type
					if !types.Implements(obj.Type(), iface) {
						continue
					}
					ms := types.NewMethodSet(obj.Type())
					n := ms.Len()
					for i := 0; i < n; i++ {
						meth := ms.At(i).Obj().(*types.Func)
						m := iface.NumMethods()
						found := false
						for j := 0; j < m; j++ {
							if iface.Method(j).Name() == meth.Name() {
								found = true
								break
							}
						}
						if !found {
							continue
						}
						c.graph.markUsedBy(meth, iface)
					}
				}
			}
		}

		fn := func(node1 ast.Node) bool {
			if node1 == nil {
				return false
			}

			if decl, ok := node1.(*ast.GenDecl); ok {
				for _, spec := range decl.Specs {
					spec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range spec.Names {
						if i >= len(spec.Values) {
							break
						}
						value := spec.Values[i]
						c.graph.markUsedBy(c.pkg.TypeOf(value), c.pkg.ObjectOf(name))
					}
				}
			}

			expr, ok := node1.(ast.Expr)
			if !ok {
				return true
			}

			left := c.pkg.TypeOf(expr)
			if left == nil {
				return true
			}
			fn2 := func(node2 ast.Node) bool {
				if node2 == nil || node1 == node2 {
					return true
				}
				switch node2 := node2.(type) {
				case *ast.Ident:
					right := c.pkg.ObjectOf(node2)
					if right == nil {
						return true
					}
					c.graph.markUsedBy(right, left)
				case ast.Expr:
					right := c.pkg.TypeOf(expr)
					if right == nil {
						return true
					}
					c.graph.markUsedBy(right, left)
				}

				return true
			}
			ast.Inspect(node1, fn2)
			return true
		}
		for _, file := range c.pkg.Files {
			ast.Inspect(file, fn)
		}
	}

	for _, node := range c.graph.nodes {
		obj, ok := node.obj.(types.Object)
		if !ok {
			continue
		}
		typNode, ok := c.graph.nodes[obj.Type()]
		if !ok {
			continue
		}
		node.uses[typNode] = struct{}{}
	}

	roots := map[*graphNode]struct{}{}
	for _, root := range c.graph.roots {
		roots[root] = struct{}{}
	}
	markNodesUsed(roots, 0)
	c.graph.markNodesQuiet()

	for _, node := range c.graph.nodes {
		if node.used || node.quiet {
			continue
		}
		found := false
		if !false {
			for _, pkg := range c.lprog.InitialPackages() {
				obj, ok := node.obj.(types.Object)
				if !ok {
					continue
				}
				if pkg.Pkg == obj.Pkg() {
					found = true
					break
				}
			}
		}
		if !found {
			continue
		}
		// FIXME ignore stdlib (unless we're testing stdlib) and vendor
		obj, ok := node.obj.(types.Object)
		if !ok {
			continue
		}
		pos := c.lprog.Fset.Position(obj.Pos())
		unused = append(unused, Unused{Obj: obj, Position: pos})
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

func (c *Checker) isRoot(obj types.Object, wholeProgram bool) bool {
	// - in local mode, main, init, tests, and non-test, non-main exported are roots
	// - in global mode (not yet implemented), main, init and tests are roots

	// FIXME consider interfaces here?

	if _, ok := obj.(*types.PkgName); ok {
		return true
	}

	if isMain(obj) || (isFunction(obj) && !isMethod(obj) && obj.Name() == "init") {
		return true
	}
	if obj.Exported() {
		// FIXME fields are only roots if the struct type would be, too
		// FIXME exported methods on unexported types aren't roots
		if (isMethod(obj) || isField(obj)) && !wholeProgram {
			return true
		}

		f := c.lprog.Fset.Position(obj.Pos()).Filename
		if strings.HasSuffix(f, "_test.go") {
			return strings.HasPrefix(obj.Name(), "Test") ||
				strings.HasPrefix(obj.Name(), "Benchmark") ||
				strings.HasPrefix(obj.Name(), "Example")
		}

		// Package-level are used, except in package main
		if isPkgScope(obj) && obj.Pkg().Name() != "main" && !wholeProgram {
			return true
		}
	}
	return false
}

func markNodesUsed(nodes map[*graphNode]struct{}, n int) {
	for node := range nodes {
		log.Printf("%s%s", strings.Repeat("\t", n), node.obj)
		wasUsed := node.used
		node.used = true
		if !wasUsed {
			markNodesUsed(node.uses, n+1)
		}
	}
}

func (g *graph) markNodesQuiet() {
	for _, node := range g.nodes {
		if node.used {
			continue
		}
		switch obj := node.obj.(type) {
		case *types.Struct:
			n := obj.NumFields()
			for i := 0; i < n; i++ {
				field := obj.Field(i)
				g.nodes[field].quiet = true
			}
		case *types.Scope:
			if obj == nil {
				continue
			}
			n := obj.NumChildren()
			for i := 0; i < n; i++ {
				scope := obj.Child(i)
				for _, name := range scope.Names() {
					v := scope.Lookup(name)
					g.nodes[v].quiet = true
				}
			}
		}
	}
}
