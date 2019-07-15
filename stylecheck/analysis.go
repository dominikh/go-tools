package stylecheck

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"honnef.co/go/tools/config"
	"honnef.co/go/tools/facts"
	"honnef.co/go/tools/internal/passes/buildssa"
	"honnef.co/go/tools/lint/lintutil"
)

var Analyzers = lintutil.InitializeAnalyzers(Docs, map[string]*analysis.Analyzer{
	"ST1000": {
		Run: CheckPackageComment,
	},
	"ST1001": {
		Run:      CheckDotImports,
		Requires: []*analysis.Analyzer{facts.Generated, config.Analyzer},
	},
	"ST1003": {
		Run:      CheckNames,
		Requires: []*analysis.Analyzer{facts.Generated, config.Analyzer},
	},
	"ST1005": {
		Run:      CheckErrorStrings,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"ST1006": {
		Run:      CheckReceiverNames,
		Requires: []*analysis.Analyzer{buildssa.Analyzer, facts.Generated},
	},
	"ST1008": {
		Run:      CheckErrorReturn,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"ST1011": {
		Run: CheckTimeNames,
	},
	"ST1012": {
		Run:      CheckErrorVarNames,
		Requires: []*analysis.Analyzer{config.Analyzer},
	},
	"ST1013": {
		Run:      CheckHTTPStatusCodes,
		Requires: []*analysis.Analyzer{facts.Generated, facts.TokenFile, config.Analyzer},
	},
	"ST1015": {
		Run:      CheckDefaultCaseOrder,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated, facts.TokenFile},
	},
	"ST1016": {
		Run:      CheckReceiverNamesIdentical,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"ST1017": {
		Run:      CheckYodaConditions,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated, facts.TokenFile},
	},
	"ST1018": {
		Run:      CheckInvisibleCharacters,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"ST1019": {
		Run:      CheckDuplicatedImports,
		Requires: []*analysis.Analyzer{facts.Generated, config.Analyzer},
	},
})
