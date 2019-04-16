package simple

import (
	"flag"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"honnef.co/go/tools/internal/passes/buildssa"
	"honnef.co/go/tools/lint"
	"honnef.co/go/tools/lint/lintutil"
)

func newFlagSet() flag.FlagSet {
	fs := flag.NewFlagSet("", flag.PanicOnError)
	fs.Var(lintutil.NewVersionFlag(), "go", "Target Go version")
	return *fs
}

var Analyzers = map[string]*analysis.Analyzer{
	"S1000": {
		Name:     "S1000",
		Run:      LintSingleCaseSelect,
		Doc:      docS1000,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1001": {
		Name:     "S1001",
		Run:      LintLoopCopy,
		Doc:      docS1001,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1002": {
		Name:     "S1002",
		Run:      LintIfBoolCmp,
		Doc:      docS1002,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1003": {
		Name:     "S1003",
		Run:      LintStringsContains,
		Doc:      docS1003,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1004": {
		Name:     "S1004",
		Run:      LintBytesCompare,
		Doc:      docS1004,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1005": {
		Name:     "S1005",
		Run:      LintUnnecessaryBlank,
		Doc:      docS1005,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1006": {
		Name:     "S1006",
		Run:      LintForTrue,
		Doc:      docS1006,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1007": {
		Name:     "S1007",
		Run:      LintRegexpRaw,
		Doc:      docS1007,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1008": {
		Name:     "S1008",
		Run:      LintIfReturn,
		Doc:      docS1008,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1009": {
		Name:     "S1009",
		Run:      LintRedundantNilCheckWithLen,
		Doc:      docS1009,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1010": {
		Name:     "S1010",
		Run:      LintSlicing,
		Doc:      docS1010,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1011": {
		Name:     "S1011",
		Run:      LintLoopAppend,
		Doc:      docS1011,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1012": {
		Name:     "S1012",
		Run:      LintTimeSince,
		Doc:      docS1012,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1016": {
		Name:     "S1016",
		Run:      LintSimplerStructConversion,
		Doc:      docS1016,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1017": {
		Name:     "S1017",
		Run:      LintTrim,
		Doc:      docS1017,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1018": {
		Name:     "S1018",
		Run:      LintLoopSlide,
		Doc:      docS1018,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1019": {
		Name:     "S1019",
		Run:      LintMakeLenCap,
		Doc:      docS1019,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1020": {
		Name:     "S1020",
		Run:      LintAssertNotNil,
		Doc:      docS1020,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1021": {
		Name:     "S1021",
		Run:      LintDeclareAssign,
		Doc:      docS1021,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1023": {
		Name:     "S1023",
		Run:      LintRedundantBreak,
		Doc:      docS1023,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1024": {
		Name:     "S1024",
		Run:      LintTimeUntil,
		Doc:      docS1024,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1025": {
		Name:     "S1025",
		Run:      LintRedundantSprintf,
		Doc:      docS1025,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1028": {
		Name:     "S1028",
		Run:      LintErrorsNewSprintf,
		Doc:      docS1028,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1029": {
		Name:     "S1029",
		Run:      LintRangeStringRunes,
		Doc:      docS1029,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
		Flags:    newFlagSet(),
	},
	"S1030": {
		Name:     "S1030",
		Run:      LintBytesBufferConversions,
		Doc:      docS1030,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1031": {
		Name:     "S1031",
		Run:      LintNilCheckAroundRange,
		Doc:      docS1031,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1032": {
		Name:     "S1032",
		Run:      LintSortHelpers,
		Doc:      docS1032,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1033": {
		Name:     "S1033",
		Run:      LintGuardedDelete,
		Doc:      docS1033,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
	"S1034": {
		Name:     "S1034",
		Run:      LintSimplifyTypeSwitch,
		Doc:      docS1034,
		Requires: []*analysis.Analyzer{inspect.Analyzer, lint.IsGeneratedAnalyzer},
		Flags:    newFlagSet(),
	},
}
