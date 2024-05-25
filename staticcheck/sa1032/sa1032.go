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
	Doc: &lint.RawDocumentation{
		Title: `Wrong order of arguments to \'errors.Is\' or \'errors.As\'`,
		Text: `
The first argument of the functions \'errors.As\' and \'errors.Is\' is the error
that we have and the second argument is the error we're trying to match against.
For example:

	if errors.Is(err, io.EOF) { ... }

This check detects some cases where the two arguments have been swapped. It
flags any calls where the first argument is referring to a package-level error
variable, such as

	if errors.Is(io.EOF, err) { /* this is wrong */ }`,
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
