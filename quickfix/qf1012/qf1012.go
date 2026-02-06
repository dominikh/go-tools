package qf1012

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/edit"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/knowledge"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "QF1012",
		Run:      run,
		Requires: code.RequiredAnalyzers,
	},
	Doc: &lint.RawDocumentation{
		Title:    `Use \'fmt.Fprintf(x, ...)\' instead of \'x.Write(fmt.Sprintf(...))\'`,
		Since:    "2022.1",
		Severity: lint.SeverityHint,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var (
	checkWriteBytesSprintfQ = pattern.MustParse(`
	(CallExpr
		(SelectorExpr recv (Ident "Write"))
		(CallExpr (ArrayType nil (Ident "byte"))
			(CallExpr
				fn@(Or
					(Symbol "fmt.Sprint")
					(Symbol "fmt.Sprintf")
					(Symbol "fmt.Sprintln"))
				args)
	))`)

	checkWriteStringSprintfQ = pattern.MustParse(`
	(CallExpr
		(SelectorExpr recv (Ident "WriteString"))
		(CallExpr
			fn@(Or
				(Symbol "fmt.Sprint")
				(Symbol "fmt.Sprintf")
				(Symbol "fmt.Sprintln"))
			args))`)

	checkWriteStringConcatQ = pattern.MustParse(`
	(CallExpr
		(SelectorExpr recv (Ident "WriteString"))
		concat@(BinaryExpr _ "+" (BasicLit "STRING" _)))`)
)

func run(pass *analysis.Pass) (any, error) {
	fn := func(node ast.Node) {
		getRecv := func(m *pattern.Matcher) (ast.Expr, ast.Expr, types.Type) {
			recv := m.State["recv"].(ast.Expr)
			recvT := pass.TypesInfo.TypeOf(recv)

			// Use *N, not N, for the interface check if N
			// is a named non-interface type, since the pointer
			// has a larger method set (https://staticcheck.dev/issues/1097).
			// We assume the receiver expression is addressable
			// since otherwise the code wouldn't compile.
			if _, ok := types.Unalias(recvT).(*types.Named); ok && !types.IsInterface(recvT) {
				recvT = types.NewPointer(recvT)
				recvPtr := &ast.UnaryExpr{Op: token.AND, X: recv}
				return recvPtr, recv, recvT
			}
			return recv, recv, recvT
		}

		if m, ok := code.Match(pass, checkWriteBytesSprintfQ, node); ok {
			recvPtr, _, recvT := getRecv(m)
			if !types.Implements(recvT, knowledge.Interfaces["io.Writer"]) {
				return
			}

			name := m.State["fn"].(*types.Func).Name()
			newName := "F" + strings.TrimPrefix(name, "S")
			msg := fmt.Sprintf("Use fmt.%s(...) instead of Write([]byte(fmt.%s(...)))", newName, name)

			args := m.State["args"].([]ast.Expr)
			fix := edit.Fix(msg, edit.ReplaceWithNode(pass.Fset, node, &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("fmt"),
					Sel: ast.NewIdent(newName),
				},
				Args: append([]ast.Expr{recvPtr}, args...),
			}))
			report.Report(pass, node, msg, report.Fixes(fix))
		} else if m, ok := code.Match(pass, checkWriteStringSprintfQ, node); ok {
			recvPtr, _, recvT := getRecv(m)
			if !types.Implements(recvT, knowledge.Interfaces["io.StringWriter"]) {
				return
			}
			// The type needs to implement both StringWriter and Writer.
			// If it doesn't implement Writer, then we cannot pass it to fmt.Fprint.
			if !types.Implements(recvT, knowledge.Interfaces["io.Writer"]) {
				return
			}

			name := m.State["fn"].(*types.Func).Name()
			newName := "F" + strings.TrimPrefix(name, "S")
			msg := fmt.Sprintf("Use fmt.%s(...) instead of WriteString(fmt.%s(...))", newName, name)

			args := m.State["args"].([]ast.Expr)
			fix := edit.Fix(msg, edit.ReplaceWithNode(pass.Fset, node, &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("fmt"),
					Sel: ast.NewIdent(newName),
				},
				Args: append([]ast.Expr{recvPtr}, args...),
			}))
			report.Report(pass, node, msg, report.Fixes(fix))
		} else if m, ok := code.Match(pass, checkWriteStringConcatQ, node); ok {
			_, recv, recvT := getRecv(m)
			// TODO(IlyasYOY): add support for []bytes, similar to the block above.
			if !types.Implements(recvT, knowledge.Interfaces["io.StringWriter"]) {
				return
			}

			concat := m.State["concat"].(ast.Expr)
			concatLits := unwrapRight(concat)

			editStmts := make([]ast.Stmt, 0, len(concatLits))
			for _, lit := range concatLits {
				editStmts = append(editStmts, &ast.ExprStmt{
					X: &ast.CallExpr{
						Fun:  &ast.SelectorExpr{X: recv, Sel: ast.NewIdent("WriteString")},
						Args: []ast.Expr{lit},
					},
				})
			}

			const msg = "Replace WriteString(x + y) with WriteString(x); WriteString(y)"
			fix := edit.Fix(msg, edit.ReplaceWithStatements(pass.Fset, node, editStmts...))
			report.Report(pass, node, msg, report.Fixes(fix))
		}
	}
	if !code.CouldMatchAny(pass, checkWriteBytesSprintfQ, checkWriteStringSprintfQ, checkWriteStringConcatQ) {
		return nil, nil
	}
	code.Preorder(pass, fn, (*ast.CallExpr)(nil))
	return nil, nil
}

func unwrapRight(rightExpr ast.Expr) []*ast.BasicLit {
	rightExpr = ast.Unparen(rightExpr)

	if bin, ok := rightExpr.(*ast.BinaryExpr); ok {
		if bin.Op != token.ADD {
			return nil
		}

		xs := unwrapRight(bin.X)
		ys := unwrapRight(bin.Y)

		all := make([]*ast.BasicLit, 0, len(xs)+len(ys))
		all = append(all, xs...)
		all = append(all, ys...)

		return all
	}

	// TODO(IlyasYOY): handle case for any expression returning string.
	if lit, ok := rightExpr.(*ast.BasicLit); ok {
		if lit.Kind == token.STRING {
			return []*ast.BasicLit{lit}
		}
	}

	return nil
}
