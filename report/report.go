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
	"honnef.co/go/tools/ir"
	"honnef.co/go/tools/lint"
)

type Options struct {
	ShortRange      bool
	FilterGenerated bool
	Fixes           []analysis.SuggestedFix
	Related         []analysis.RelatedInformation
}

type fullPositioner interface {
	Pos() token.Pos
	End() token.Pos
}

type Option func(*Options)

func ShortRange() Option {
	return func(opts *Options) {
		opts.ShortRange = true
	}
}

func FilterGenerated() Option {
	return func(opts *Options) {
		opts.FilterGenerated = true
	}
}

func Fixes(fixes ...analysis.SuggestedFix) Option {
	return func(opts *Options) {
		opts.Fixes = append(opts.Fixes, fixes...)
	}
}

func Related(node lint.Positioner, message string) Option {
	return func(opts *Options) {
		start, end := nodeRange(node)
		r := analysis.RelatedInformation{
			Pos:     start,
			End:     end,
			Message: message,
		}
		opts.Related = append(opts.Related, r)
	}
}

func nodeRange(node lint.Positioner) (start, end token.Pos) {
	if irnode, ok := node.(ir.Node); ok {
		if refs := irnode.Referrers(); refs != nil {
			for _, ref := range *refs {
				if ref, ok := ref.(*ir.DebugRef); ok {
					node = ref.Expr
					break
				}
			}
		}
	}
	switch node := node.(type) {
	case fullPositioner:
		start = node.Pos()
		end = node.End()
	default:
		start = node.Pos()
		end = token.NoPos
	}
	return start, end
}

func Report(pass *analysis.Pass, node lint.Positioner, message string, opts ...Option) {
	start, end := nodeRange(node)
	cfg := &Options{}
	for _, opt := range opts {
		opt(cfg)
	}

	file := lint.DisplayPosition(pass.Fset, start).Filename
	if cfg.FilterGenerated {
		m := pass.ResultOf[facts.Generated].(map[string]facts.Generator)
		if _, ok := m[file]; ok {
			return
		}
	}

	if cfg.ShortRange {
		switch node := node.(type) {
		case *ast.IfStmt:
			end = node.Cond.End()
		case *ast.File:
			end = node.Name.End()
		default:
			panic(fmt.Sprintf("unhandled type %T", node))
		}
	}

	d := analysis.Diagnostic{
		Pos:            start,
		End:            end,
		Message:        message,
		SuggestedFixes: cfg.Fixes,
		Related:        cfg.Related,
	}
	pass.Report(d)
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
