package sa9009

import (
	"go/ast"
	"go/types"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA9009",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	Doc: &lint.Documentation{
		Title: `Returned function should be called in defer`,
		Text: `
If you have a function such as:

    func f() func() {
        // Do something.
        return func() {
            // Do something.
        }
    }

Then calling that in defer:

    defer f()

Is almost always a mistake, since you typically want to call the returned
function:

    defer f()()
`,
		Since:    "Unreleased",
		Severity: lint.SeverityWarning,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

func run(pass *analysis.Pass) (any, error) {
	checkIdent := func(c *ast.Ident) bool {
		var (
			obj = pass.TypesInfo.ObjectOf(c)
			sig *types.Signature
		)
		switch f := obj.(type) {
		case *types.Builtin:
			return false
		case *types.Func:
			sig = f.Type().(*types.Signature)
		case *types.Var:
			switch ff := f.Type().(type) {
			case *types.Signature:
				sig = ff
			case *types.Named:
				sig = ff.Underlying().(*types.Signature)
			}
		}
		r := sig.Results()
		if r != nil && r.Len() == 1 {
			_, ok := r.At(0).Type().(*types.Signature)
			return ok
		}
		return false
	}

	fn := func(n ast.Node) {
		var (
			returnsFunc bool
			def         = n.(*ast.DeferStmt)
		)
		switch c := def.Call.Fun.(type) {
		case *ast.FuncLit: // defer func() { }()
			r := c.Type.Results
			if r != nil && len(r.List) == 1 {
				_, returnsFunc = r.List[0].Type.(*ast.FuncType)
			}
		case *ast.Ident: // defer f()
			returnsFunc = checkIdent(c)
		case *ast.SelectorExpr: // defer t.f()
			returnsFunc = checkIdent(c.Sel)
		case *ast.IndexExpr: // defer f[int](0)
			if id, ok := c.X.(*ast.Ident); ok {
				returnsFunc = checkIdent(id)
			}
		}
		if returnsFunc {
			report.Report(pass, def, "defered return function not called")
		}
	}

	code.Preorder(pass, fn, (*ast.DeferStmt)(nil))
	return nil, nil
}
