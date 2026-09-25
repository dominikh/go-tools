package st1023

import (
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/internal/sharedcheck"
)

func init() {
	SCAnalyzer.Analyzer.Name = "ST1023"
}

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: sharedcheck.RedundantTypeInDeclarationChecker("should", false),
	Doc: &lint.RawDocumentation{
		Title: "Redundant type in variable declaration",
		Text: `In a local variable declaration, omit an explicit type when the
initializer already determines the same type. For example, use
\'var x = f()\' instead of \'var x T = f()\' when \'f\' returns \'T\'. Keep the
explicit type when omitting it would change the variable's type, such
as when assigning a concrete value to an interface variable.

This check does not flag redundant types in packages that import
syscall or unsafe. Such packages sometimes declare variables with an
explicit type to make sure a value has exactly the expected type.`,
		Since:      "2021.1",
		NonDefault: true,
		MergeIf:    lint.MergeIfAll,
	},
})

var Analyzer = SCAnalyzer.Analyzer
