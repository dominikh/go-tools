package quickfix

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/edit"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/go/types/typeutil"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
)

var (
	stringsIndexQ = pattern.MustParse(`
		(CallExpr
			fn@(Or
				(Function "strings.Index")
				(Function "strings.LastIndex")
				(Function "strings.IndexByte")
				(Function "strings.LastIndexByte"))
			[arg1 lit@(BasicLit (Or "STRING" "CHAR") _)])`)
	bytesIndexQ = pattern.MustParse(`
		(CallExpr
			fn@(Or
				(Function "bytes.Index")
				(Function "bytes.LastIndex"))
			[
				arg1
				(Or
					(CompositeLit _ [b])
					(CallExpr (ArrayType nil _) lit@(BasicLit "STRING" _)))])`)

	stringsIndexR     = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "strings") (Ident replacement)) [arg1 (BasicLit "CHAR" lit)])`)
	stringsIndexByteR = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "strings") (Ident replacement)) [arg1 (BasicLit "STRING" lit)])`)
	bytesIndexR       = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "bytes") (Ident replacement)) [arg1 b])`)
)

func CheckStringsIndexByte(pass *analysis.Pass) (interface{}, error) {
	// FIXME(dh): create proper suggested fix for renamed import

	fn := func(node ast.Node) {
		if matcher, ok := code.Match(pass, stringsIndexQ, node); ok {
			var replacement string
			isString := false
			fn := typeutil.FuncName(matcher.State["fn"].(*types.Func))
			switch fn {
			case "strings.Index":
				replacement = "IndexByte"
				isString = true
			case "strings.LastIndex":
				replacement = "LastIndexByte"
				isString = true
			case "strings.IndexByte":
				replacement = "Index"
			case "strings.LastIndexByte":
				replacement = "LastIndex"
			}

			lit := matcher.State["lit"].(*ast.BasicLit).Value
			var newLit string
			if isString {
				if q, err := strconv.Unquote(lit); err != nil || len(q) != 1 {
					return
				}
				newLit = "'" + lit[1:len(lit)-1] + "'"
			} else {
				newLit = `"` + lit[1:len(lit)-1] + `"`
			}

			state := pattern.State{
				"arg1":        matcher.State["arg1"],
				"lit":         newLit,
				"replacement": replacement,
			}
			report.Report(pass, node, fmt.Sprintf("could use strings.%s instead of %s", replacement, fn),
				report.Fixes(edit.Fix(fmt.Sprintf("Use strings.%s instead of %s", replacement, fn), edit.ReplaceWithPattern(pass, stringsIndexR, state, node))))
		} else if matcher, ok := code.Match(pass, bytesIndexQ, node); ok {
			var replacement string
			fn := typeutil.FuncName(matcher.State["fn"].(*types.Func))
			switch fn {
			case "bytes.Index":
				replacement = "IndexByte"
			case "bytes.LastIndex":
				replacement = "LastIndexByte"
			}

			if _, ok := matcher.State["b"]; ok {
				state := pattern.State{
					"arg1":        matcher.State["arg1"],
					"b":           matcher.State["b"],
					"replacement": replacement,
				}
				report.Report(pass, node, fmt.Sprintf("could use bytes.%s instead of %s", replacement, fn),
					report.Fixes(edit.Fix(fmt.Sprintf("Use bytes.%s instead of %s", replacement, fn), edit.ReplaceWithPattern(pass, bytesIndexR, state, node))))
			} else {
				lit := matcher.State["lit"].(*ast.BasicLit).Value
				if q, err := strconv.Unquote(lit); err != nil || len(q) != 1 {
					return
				}
				state := pattern.State{
					"arg1": matcher.State["arg1"],
					"b": &ast.BasicLit{
						Kind:  token.CHAR,
						Value: "'" + lit[1:len(lit)-1] + "'",
					},
					"replacement": replacement,
				}
				report.Report(pass, node, fmt.Sprintf("could use bytes.%s instead of %s", replacement, fn),
					report.Fixes(edit.Fix(fmt.Sprintf("Use bytes.%s instead of %s", replacement, fn), edit.ReplaceWithPattern(pass, bytesIndexR, state, node))))
			}
		}
	}
	code.Preorder(pass, fn, (*ast.CallExpr)(nil))
	return nil, nil
}
