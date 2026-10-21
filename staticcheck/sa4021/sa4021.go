package sa4021

import (
	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/facts/generated"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA4021",
		Run:      run,
		Requires: append([]*analysis.Analyzer{generated.Analyzer}, code.RequiredAnalyzers...),
	},
	Doc: &lint.RawDocumentation{
		Title: `\"x = append(y)\" is equivalent to \"x = y\"`,
		Text: `Calling \'append\' with only a slice and no elements returns the slice
unchanged. It does not copy its elements or allocate a new backing
array. Use the slice directly if sharing its backing array is intended,
or use \'slices.Clone\' to make a shallow copy. If elements were meant to
be appended, supply them as additional arguments.`,
		Since:    "2019.2",
		Severity: lint.SeverityWarning,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var checkSingleArgAppendQ = pattern.MustParse(`(CallExpr (Builtin "append") [_])`)

func run(pass *analysis.Pass) (any, error) {
	for node := range code.Matches(pass, checkSingleArgAppendQ) {
		report.Report(pass, node, "x = append(y) is equivalent to x = y", report.FilterGenerated())
	}
	return nil, nil
}
