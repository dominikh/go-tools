package code

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"slices"

	typeindexanalyzer "honnef.co/go/tools/internal/analysisinternal/typeindex"
	"honnef.co/go/tools/internal/typesinternal/typeindex"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func Preorder(pass *analysis.Pass, fn func(ast.Node), types ...ast.Node) {
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder(types, fn)
}

func PreorderStack(pass *analysis.Pass, fn func(ast.Node, []ast.Node), types ...ast.Node) {
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).WithStack(types, func(n ast.Node, push bool, stack []ast.Node) (proceed bool) {
		if push {
			fn(n, stack)
		}
		return true
	})
}

func Match(pass *analysis.Pass, q pattern.Pattern, node ast.Node) (*pattern.Matcher, bool) {
	// Note that we ignore q.Relevant – callers of Match usually use
	// AST inspectors that already filter on nodes we're interested
	// in.
	m := &pattern.Matcher{TypesInfo: pass.TypesInfo}
	ok := m.Match(q, node)
	return m, ok
}

func CouldMatch(pass *analysis.Pass, q pattern.Pattern) bool {
	index := pass.ResultOf[typeindexanalyzer.Analyzer].(*typeindex.Index)
	var do func(node pattern.Node) bool
	do = func(node pattern.Node) bool {
		switch node := node.(type) {
		case pattern.Any:
			return true
		case pattern.Or:
			return slices.ContainsFunc(node.Nodes, do)
		case pattern.And:
			for _, child := range node.Nodes {
				if !do(child) {
					return false
				}
			}
			return true
		case pattern.IndexSymbol:
			if node.Type == "" {
				return index.Object(node.Path, node.Ident) != nil
			} else {
				return index.Selection(node.Path, node.Type, node.Ident) != nil
			}
		default:
			panic(fmt.Sprintf("internal error: unexpected type %T", node))
		}
	}
	return do(q.SymbolsPattern)
}

func MatchAndEdit(pass *analysis.Pass, before, after pattern.Pattern, node ast.Node) (*pattern.Matcher, []analysis.TextEdit, bool) {
	m, ok := Match(pass, before, node)
	if !ok {
		return m, nil, false
	}
	r := pattern.NodeToAST(after.Root, m.State)
	buf := &bytes.Buffer{}
	format.Node(buf, pass.Fset, r)
	edit := []analysis.TextEdit{{
		Pos:     node.Pos(),
		End:     node.End(),
		NewText: buf.Bytes(),
	}}
	return m, edit, true
}
