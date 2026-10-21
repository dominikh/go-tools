package qf1011

import (
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/internal/sharedcheck"
)

func init() {
	SCAnalyzer.Analyzer.Name = "QF1011"
}

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: sharedcheck.RedundantTypeInDeclarationChecker("could", true),
	Doc: &lint.RawDocumentation{
		Title: "Omit redundant type from variable declaration",
		Text: `A local variable's type can be inferred from its initial value. When
this inferred type is identical to the explicitly declared type, the
explicit type can be omitted to make the declaration shorter without
changing its meaning. For example, \'var x int = 1\' can be written as
\'var x = 1\'.`,
		Since:    "2021.1",
		Severity: lint.SeverityHint,
	},
})

var Analyzer = SCAnalyzer.Analyzer
