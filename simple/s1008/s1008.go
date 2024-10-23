package s1008

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"strings"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/facts/generated"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "S1008",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer, generated.Analyzer},
	},
	Doc: &lint.RawDocumentation{
		Title: `Simplify returning boolean expression`,
		Before: `
if <expr> {
    return true
}
return false`,
		After:   `return <expr>`,
		Since:   "2017.1",
		MergeIf: lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var (
	checkIfReturnQIf = pattern.MustParse(`
		(IfStmt 
			nil 
			cond 
			[fullret@(ReturnStmt _)] 
			nil)
	`)
	checkIfReturnQRet = pattern.MustParse(`
		(Binding "fullret" (ReturnStmt _))
	`)
	checkReturnValue = pattern.MustParse(`
		(ReturnStmt 
			(List 
				ret@(Builtin (Or "true" "false"))
				tail@(Any)))
	`)
)

func run(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		block := node.(*ast.BlockStmt)
		l := len(block.List)
		if l < 2 {
			return
		}
		n1, n2 := block.List[l-2], block.List[l-1]

		if len(block.List) >= 3 {
			if _, ok := block.List[l-3].(*ast.IfStmt); ok {
				// Do not flag a series of if statements
				return
			}
		}
		fullm1, ok := code.Match(pass, checkIfReturnQIf, n1)
		if !ok {
			return
		}
		fullm2, ok := code.Match(pass, checkIfReturnQRet, n2)
		if !ok {
			return
		}

		if op, ok := fullm1.State["cond"].(*ast.BinaryExpr); ok {
			switch op.Op {
			case token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
			default:
				return
			}
		}

		fullret1 := fullm1.State["fullret"].(*ast.ReturnStmt)
		m1, ok := code.Match(pass, checkReturnValue, fullret1)
		if !ok {
			return
		}

		fullret2 := fullm2.State["fullret"].(*ast.ReturnStmt)
		m2, ok := code.Match(pass, checkReturnValue, fullret2)
		if !ok {
			return
		}

		ret1, tail1 := getRetAndTail(m1)
		tail1String := renderTailString(pass, tail1)

		ret2, tail2 := getRetAndTail(m2)
		tail2String := renderTailString(pass, tail2)

		if tail1String != tail2String {
			// we want to process only return with the same values
			return
		}

		if ret1.Name == ret2.Name {
			// we want the function to return true and false, not the
			// same value both times.
			return
		}

		cond := fullm1.State["cond"].(ast.Expr)
		origCond := cond
		if ret1.Name == "false" {
			cond = negate(pass, cond)
		}
		report.Report(pass, n1,
			fmt.Sprintf(
				"should use 'return %s%s' instead of 'if %s { return %s%s }; return %s%s'",
				report.Render(pass, cond),
				tail1String,
				report.Render(
					pass,
					origCond,
				),
				report.Render(pass, ret1),
				tail1String,
				report.Render(pass, ret2),
				tail2String,
			),
			report.FilterGenerated())
	}
	code.Preorder(pass, fn, (*ast.BlockStmt)(nil))
	return nil, nil
}

func getRetAndTail(m *pattern.Matcher) (*ast.Ident, []ast.Expr) {
	ret1 := m.State["ret"].(*ast.Ident)
	var tail []ast.Expr
	if t, ok := m.State["tail"]; ok {
		tail, _ = t.([]ast.Expr)
	}
	return ret1, tail
}

func renderTailString(pass *analysis.Pass, tail []ast.Expr) string {
	var tailStringBuilder strings.Builder
	if len(tail) != 0 {
		tailStringBuilder.WriteString(", ")
		tailStringBuilder.WriteString(report.RenderArgs(pass, tail))
	}
	return tailStringBuilder.String()
}

func negate(pass *analysis.Pass, expr ast.Expr) ast.Expr {
	switch expr := expr.(type) {
	case *ast.BinaryExpr:
		out := *expr
		switch expr.Op {
		case token.EQL:
			out.Op = token.NEQ
		case token.LSS:
			out.Op = token.GEQ
		case token.GTR:
			// Some builtins never return negative ints; "len(x) <= 0" should be "len(x) == 0".
			if call, ok := expr.X.(*ast.CallExpr); ok &&
				code.IsCallToAny(pass, call, "len", "cap", "copy") &&
				code.IsIntegerLiteral(pass, expr.Y, constant.MakeInt64(0)) {
				out.Op = token.EQL
			} else {
				out.Op = token.LEQ
			}
		case token.NEQ:
			out.Op = token.EQL
		case token.LEQ:
			out.Op = token.GTR
		case token.GEQ:
			out.Op = token.LSS
		}
		return &out
	case *ast.Ident, *ast.CallExpr, *ast.IndexExpr, *ast.StarExpr:
		return &ast.UnaryExpr{
			Op: token.NOT,
			X:  expr,
		}
	case *ast.UnaryExpr:
		if expr.Op == token.NOT {
			return expr.X
		}
		return &ast.UnaryExpr{
			Op: token.NOT,
			X:  expr,
		}
	default:
		return &ast.UnaryExpr{
			Op: token.NOT,
			X: &ast.ParenExpr{
				X: expr,
			},
		}
	}
}
