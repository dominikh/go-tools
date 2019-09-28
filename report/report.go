package report

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/facts"
	"honnef.co/go/tools/lint"
)

type Options struct {
	Node            ast.Node
	ShortRange      bool
	FilterGenerated bool
	Message         string
	Fixes           []analysis.SuggestedFix
}

func Report(pass *analysis.Pass, opts Options) {
	file := lint.DisplayPosition(pass.Fset, opts.Node.Pos()).Filename
	if opts.FilterGenerated {
		m := pass.ResultOf[facts.Generated].(map[string]facts.Generator)
		if _, ok := m[file]; ok {
			return
		}
	}

	start := opts.Node.Pos()
	end := opts.Node.End()
	if opts.ShortRange {
		switch node := opts.Node.(type) {
		case *ast.IfStmt:
			end = node.Cond.End()
		default:
			panic(fmt.Sprintf("unhandled type %T", node))
		}
	}

	d := analysis.Diagnostic{
		Pos:            start,
		End:            end,
		Message:        opts.Message,
		SuggestedFixes: opts.Fixes,
	}
	pass.Report(d)
}

func PosfFG(pass *analysis.Pass, pos token.Pos, f string, args ...interface{}) {
	file := lint.DisplayPosition(pass.Fset, pos).Filename
	m := pass.ResultOf[facts.Generated].(map[string]facts.Generator)
	if _, ok := m[file]; ok {
		return
	}
	pass.Reportf(pos, f, args...)
}

func Node(pass *analysis.Pass, node ast.Node, msg string, fixes ...analysis.SuggestedFix) {
	Report(pass, Options{
		Node:    node,
		Message: msg,
		Fixes:   fixes,
	})
}

func NodeFG(pass *analysis.Pass, node ast.Node, msg string, fixes ...analysis.SuggestedFix) {
	Report(pass, Options{
		Node:            node,
		FilterGenerated: true,
		Message:         msg,
		Fixes:           fixes,
	})
}

func Nodef(pass *analysis.Pass, node ast.Node, format string, args ...interface{}) {
	Report(pass, Options{
		Node:    node,
		Message: fmt.Sprintf(format, args...),
	})
}

func NodefFG(pass *analysis.Pass, node ast.Node, format string, args ...interface{}) {
	Report(pass, Options{
		Node:            node,
		FilterGenerated: true,
		Message:         fmt.Sprintf(format, args...),
	})
}

func Render(pass *analysis.Pass, x interface{}) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, pass.Fset, x); err != nil {
		panic(err)
	}
	return buf.String()
}

func RenderArgs(pass *analysis.Pass, args []ast.Expr) string {
	var ss []string
	for _, arg := range args {
		ss = append(ss, Render(pass, arg))
	}
	return strings.Join(ss, ", ")
}
