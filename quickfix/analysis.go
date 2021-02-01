package quickfix

import (
	"honnef.co/go/tools/analysis/lint"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var Analyzers = lint.InitializeAnalyzers(Docs, map[string]*analysis.Analyzer{
	"QF1000": {
		Run:      CheckStringsIndexByte,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"QF1001": {
		Run:      CheckDeMorgan,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"QF1002": {
		Run:      CheckTaglessSwitch,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"QF1003": {
		Run:      CheckIfElseToSwitch,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"QF1004": {
		Run:      CheckStringsReplaceAll,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"QF1005": {
		Run:      CheckMathPow,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"QF1006": {
		Run:      CheckForLoopIfBreak,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"QF1007": {
		Run:      CheckConditionalAssignment,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
})
