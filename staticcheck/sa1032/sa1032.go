package sa1032

import (
	"honnef.co/go/tools/analysis/callcheck"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/internal/passes/buildir"
	"honnef.co/go/tools/knowledge"

	"golang.org/x/tools/go/analysis"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA1032",
		Requires: []*analysis.Analyzer{buildir.Analyzer},
		Run:      callcheck.Analyzer(rules),
	},
	Doc: &lint.Documentation{
		Title: `Swapped arguments in errors.Is() or errors.As()`,
		Text: `
If the first argument to \'errors.As()\' or \'errors.Is()\' contains a package
selector it assumes the error and target were swapped, since the target should
practically always be a local variable and not a reference to another package.
`,
		Since:    "Unreleased",
		Severity: lint.SeverityError,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var rules = map[string]callcheck.Check{
	"errors.As": func(call *callcheck.Call) {
		validateIs(call.Pass.Pkg.Path(), call.Args[knowledge.Arg("errors.As.err")])
	},
	"errors.Is": func(call *callcheck.Call) {
		validateIs(call.Pass.Pkg.Path(), call.Args[knowledge.Arg("errors.Is.err")])
	},
}

func validateIs(curPkg string, arg *callcheck.Argument) {
	v, ok := arg.Value.Value.(*ir.Load)
	if !ok {
		return
	}
	g, ok := v.X.(*ir.Global)
	if !ok {
		return
	}
	if pkg := g.Package().Pkg; pkg != nil && pkg.Path() != curPkg {
		arg.Invalid("arguments have the wrong order")
	}
}
