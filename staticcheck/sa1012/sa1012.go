package sa1012

import (
	"go/ast"
	"go/types"
	"strconv"

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

var checkNilContextQ = pattern.MustParse(`(CallExpr fun@(Symbol _) (Builtin "nil"):_)`)

func run(pass *analysis.Pass) (any, error) {
	for node, m := range code.Matches(pass, checkNilContextQ) {
		call := node.(*ast.CallExpr)
		fun, ok := m.State["fun"].(*types.Func)
		if !ok {
			// it might also be a builtin
			continue
		}
		sig := fun.Type().(*types.Signature)
		if sig.Params().Len() == 0 {
			// Our CallExpr might've matched a method expression, like
			// (*T).Foo(nil) – here, nil isn't the first argument of
			// the Foo method, but the method receiver.
			continue
		}
		if !typeutil.IsTypeWithName(sig.Params().At(0).Type(), "context.Context") {
			continue
		}
		options := []report.Option(nil)
		if contextAlias := importAlias(pass, call, "context"); contextAlias != "" {
			todo := &ast.CallExpr{Fun: edit.Selector(contextAlias, "TODO")}
			bg := &ast.CallExpr{Fun: edit.Selector(contextAlias, "Background")}
			options = append(options, report.Fixes(
				edit.Fix("Use context.TODO", edit.ReplaceWithNode(pass.Fset, call.Args[0], todo)),
				edit.Fix("Use context.Background", edit.ReplaceWithNode(pass.Fset, call.Args[0], bg))))
		}
		report.Report(pass, call.Args[0],
			"do not pass a nil Context, even if a function permits it; pass context.TODO if you are unsure about which Context to use", options...)
	}
	return nil, nil
}

func importAlias(pass *analysis.Pass, node ast.Node, path string) string {
	file := code.File(pass, node)
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil || importPath != path {
			continue
		}
		var pkgName *types.PkgName
		if spec.Name == nil {
			pkgName, _ = pass.TypesInfo.Implicits[spec].(*types.PkgName)
		} else {
			pkgName, _ = pass.TypesInfo.Defs[spec.Name].(*types.PkgName)
		}
		if pkgName == nil || pkgName.Name() == "." || pkgName.Name() == "_" {
			return ""
		}
		scope := pass.TypesInfo.Scopes[file].Innermost(node.Pos())
		if scope != nil {
			_, obj := scope.LookupParent(pkgName.Name(), node.Pos())
			if obj == pkgName {
				return pkgName.Name()
			}
		}
		return ""
	}
	return ""
}
