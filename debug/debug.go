// Package debug contains helpers for debugging static analyses.
package debug

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
)

// TypeCheck parses and type-checks a single-file Go package from a string.
// The package must not have any imports.
func TypeCheck(src string) (*ast.File, *types.Package, *types.Info, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, nil, err
	}
	pkg := types.NewPackage("foo", f.Name.Name)
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Implicits:  map[ast.Node]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     map[ast.Node]*types.Scope{},
		InitOrder:  []*types.Initializer{},
		Instances:  map[*ast.Ident]types.Instance{},
	}
	tcfg := &types.Config{
		Importer: importer.Default(),
	}
	if err := types.NewChecker(tcfg, fset, pkg, info).Files([]*ast.File{f}); err != nil {
		return nil, nil, nil, err
	}
	return f, pkg, info, nil
}

func FormatNode(node ast.Node) string {
	var buf bytes.Buffer
	fset := token.NewFileSet()
	format.Node(&buf, fset, node)
	return buf.String()
}
