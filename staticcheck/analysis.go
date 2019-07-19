package staticcheck

import (
	"honnef.co/go/tools/facts"
	"honnef.co/go/tools/internal/passes/buildssa"
	"honnef.co/go/tools/lint/lintutil"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var Analyzers = lintutil.InitializeAnalyzers(Docs, map[string]*analysis.Analyzer{
	"SA1000": {
		Run:      callChecker(checkRegexpRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1001": {
		Run:      CheckTemplate,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA1002": {
		Run:      callChecker(checkTimeParseRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1003": {
		Run:      callChecker(checkEncodingBinaryRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1004": {
		Run:      CheckTimeSleepConstant,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA1005": {
		Run:      CheckExec,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA1006": {
		Run:      CheckUnsafePrintf,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA1007": {
		Run:      callChecker(checkURLsRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1008": {
		Run:      CheckCanonicalHeaderKey,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA1010": {
		Run:      callChecker(checkRegexpFindAllRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1011": {
		Run:      callChecker(checkUTF8CutsetRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1012": {
		Run:      CheckNilContext,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA1013": {
		Run:      CheckSeeker,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA1014": {
		Run:      callChecker(checkUnmarshalPointerRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1015": {
		Run:      CheckLeakyTimeTick,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA1016": {
		Run:      CheckUntrappableSignal,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA1017": {
		Run:      callChecker(checkUnbufferedSignalChanRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1018": {
		Run:      callChecker(checkStringsReplaceZeroRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1019": {
		Run:      CheckDeprecated,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Deprecated},
	},
	"SA1020": {
		Run:      callChecker(checkListenAddressRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1021": {
		Run:      callChecker(checkBytesEqualIPRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1023": {
		Run:      CheckWriterBufferModified,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA1024": {
		Run:      callChecker(checkUniqueCutsetRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1025": {
		Run:      CheckTimerResetReturnValue,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA1026": {
		Run:      callChecker(checkUnsupportedMarshal),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1027": {
		Run:      callChecker(checkAtomicAlignment),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA1028": {
		Run:      callChecker(checkSortSliceRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},

	"SA2000": {
		Run:      CheckWaitgroupAdd,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA2001": {
		Run:      CheckEmptyCriticalSection,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA2002": {
		Run:      CheckConcurrentTesting,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA2003": {
		Run:      CheckDeferLock,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},

	"SA3000": {
		Run:      CheckTestMainExit,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA3001": {
		Run:      CheckBenchmarkN,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},

	"SA4000": {
		Run:      CheckLhsRhsIdentical,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.TokenFile, facts.Generated},
	},
	"SA4001": {
		Run:      CheckIneffectiveCopy,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA4002": {
		Run:      CheckDiffSizeComparison,
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA4003": {
		Run:      CheckExtremeComparison,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA4004": {
		Run:      CheckIneffectiveLoop,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA4006": {
		Run:      CheckUnreadVariableValues,
		Requires: []*analysis.Analyzer{buildssa.Analyzer, facts.Generated},
	},
	"SA4008": {
		Run:      CheckLoopCondition,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA4009": {
		Run:      CheckArgOverwritten,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA4010": {
		Run:      CheckIneffectiveAppend,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA4011": {
		Run:      CheckScopedBreak,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA4012": {
		Run:      CheckNaNComparison,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA4013": {
		Run:      CheckDoubleNegation,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA4014": {
		Run:      CheckRepeatedIfElse,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA4015": {
		Run:      callChecker(checkMathIntRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA4016": {
		Run:      CheckSillyBitwiseOps,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.TokenFile},
	},
	"SA4017": {
		Run:      CheckPureFunctions,
		Requires: []*analysis.Analyzer{buildssa.Analyzer, facts.Purity},
	},
	"SA4018": {
		Run:      CheckSelfAssignment,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated, facts.TokenFile},
	},
	"SA4019": {
		Run:      CheckDuplicateBuildConstraints,
		Requires: []*analysis.Analyzer{facts.Generated},
	},
	"SA4020": {
		Run:      CheckUnreachableTypeCases,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA4021": {
		Run:      CheckSingleArgAppend,
		Requires: []*analysis.Analyzer{inspect.Analyzer, facts.Generated, facts.TokenFile},
	},

	"SA5000": {
		Run:      CheckNilMaps,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA5001": {
		Run:      CheckEarlyDefer,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA5002": {
		Run:      CheckInfiniteEmptyLoop,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA5003": {
		Run:      CheckDeferInInfiniteLoop,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA5004": {
		Run:      CheckLoopEmptyDefault,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA5005": {
		Run:      CheckCyclicFinalizer,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA5007": {
		Run:      CheckInfiniteRecursion,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA5008": {
		Run:      CheckStructTags,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA5009": {
		Run:      callChecker(checkPrintfRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},

	"SA6000": {
		Run:      callChecker(checkRegexpMatchLoopRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA6001": {
		Run:      CheckMapBytesKey,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA6002": {
		Run:      callChecker(checkSyncPoolValueRules),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer},
	},
	"SA6003": {
		Run:      CheckRangeStringRunes,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	},
	"SA6005": {
		Run:      CheckToLowerToUpperComparison,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},

	"SA9001": {
		Run:      CheckDubiousDeferInChannelRangeLoop,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA9002": {
		Run:      CheckNonOctalFileMode,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	"SA9003": {
		Run:      CheckEmptyBranch,
		Requires: []*analysis.Analyzer{buildssa.Analyzer, facts.TokenFile, facts.Generated},
	},
	"SA9004": {
		Run:      CheckMissingEnumTypesInDeclaration,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	// Filtering generated code because it may include empty structs generated from data models.
	"SA9005": {
		Run:      callChecker(checkNoopMarshal),
		Requires: []*analysis.Analyzer{buildssa.Analyzer, valueRangesAnalyzer, facts.Generated, facts.TokenFile},
	},

	"SA9999": {
		Run:      CheckXXX,
		Requires: []*analysis.Analyzer{ctrlflow.Analyzer},
	},
})
