package sa1012

import (
	"go/ast"
	"go/types"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/edit"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/go/types/typeutil"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA1012",
		Run:      run,
		Requires: code.RequiredAnalyzers,
	},
	Doc: &lint.RawDocumentation{
		Title: `A nil \'context.Context\' is being passed to a function, consider using \'context.TODO\' instead`,
		Text: `The context package prohibits the use of a \'nil\' context.
If no parent context is available, a new context should be used,
e.g. \'context.TODO\' or \'context.Background\'.`,
		Since:    "2017.1",
		Severity: lint.SeverityWarning,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var checkNilContextQ = pattern.MustParse(`(Or
	(CallExpr fun@(Symbol _) (Builtin "nil"):_)
	(CallExpr fun@(Symbol _) _:(Builtin "nil"):_))`)

func run(pass *analysis.Pass) (any, error) {
	todo := &ast.CallExpr{
		Fun: edit.Selector("context", "TODO"),
	}
	bg := &ast.CallExpr{
		Fun: edit.Selector("context", "Background"),
	}
	for node, m := range code.Matches(pass, checkNilContextQ) {
		call := node.(*ast.CallExpr)
		fun, ok := m.State["fun"].(*types.Func)
		if !ok {
			// it might also be a builtin
			continue
		}
		argIdx := 0
		if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
			if selection, ok := pass.TypesInfo.Selections[sel]; ok && selection.Kind() == types.MethodExpr {
				// In a method expression like (*T).Foo(recv, ctx), call.Args[0]
				// is the receiver and call.Args[1] is the method's first parameter.
				argIdx = 1
			}
		}
		if len(call.Args) <= argIdx {
			continue
		}
		arg := call.Args[argIdx]
		if id, ok := ast.Unparen(arg).(*ast.Ident); !ok || pass.TypesInfo.ObjectOf(id) != types.Universe.Lookup("nil") {
			continue
		}
		sig := fun.Type().(*types.Signature)
		if sig.Params().Len() == 0 {
			continue
		}
		if !typeutil.IsTypeWithName(sig.Params().At(0).Type(), "context.Context") {
			continue
		}
		report.Report(pass, arg,
			"do not pass a nil Context, even if a function permits it; pass context.TODO if you are unsure about which Context to use", report.Fixes(
				edit.Fix("Use context.TODO", edit.ReplaceWithNode(pass.Fset, arg, todo)),
				edit.Fix("Use context.Background", edit.ReplaceWithNode(pass.Fset, arg, bg))))
	}
	return nil, nil
}
