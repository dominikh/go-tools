package simple

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"honnef.co/go/tools/facts"
	"honnef.co/go/tools/internal/passes/buildssa"
	"honnef.co/go/tools/lint/lintutil"
)

var Analyzers = lintutil.InitializeAnalyzers(Docs, map[string]*analysis.Analyzer{
	"S1000": {
		Run:      LintSingleCaseSelect,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1001": {
		Run:      LintLoopCopy,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1002": {
		Run:      LintIfBoolCmp,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1003": {
		Run:      LintStringsContains,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1004": {
		Run:      LintBytesCompare,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1005": {
		Run:      LintUnnecessaryBlank,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1006": {
		Run:      LintForTrue,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1007": {
		Run:      LintRegexpRaw,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1008": {
		Run:      LintIfReturn,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1009": {
		Run:      LintRedundantNilCheckWithLen,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1010": {
		Run:      LintSlicing,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1011": {
		Run:      LintLoopAppend,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1012": {
		Run:      LintTimeSince,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1016": {
		Run:      LintSimplerStructConversion,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1017": {
		Run:      LintTrim,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1018": {
		Run:      LintLoopSlide,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1019": {
		Run:      LintMakeLenCap,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1020": {
		Run:      LintAssertNotNil,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1021": {
		Run:      LintDeclareAssign,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1023": {
		Run:      LintRedundantBreak,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1024": {
		Run:      LintTimeUntil,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1025": {
		Run:      LintRedundantSprintf,
		Requires: []*analysis.Analyzer{buildssa.Analyzer, inspect.Analyzer, facts.Generated},
	},
	"S1028": {
		Run:      LintErrorsNewSprintf,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1029": {
		Run:      LintRangeStringRunes,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"S1030": {
		Run:      LintBytesBufferConversions,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1031": {
		Run:      LintNilCheckAroundRange,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1032": {
		Run:      LintSortHelpers,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1033": {
		Run:      LintGuardedDelete,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1034": {
		Run:      LintSimplifyTypeSwitch,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
	"S1035": {
		Run:      LintRedundantCanonicalHeaderKey,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated},
	},
})
