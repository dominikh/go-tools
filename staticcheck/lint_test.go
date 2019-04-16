package staticcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAll(t *testing.T) {
	checks := map[string][]struct {
		dir     string
		version string
	}{
		"SA1000": {{dir: "CheckRegexps"}},
		"SA1001": {{dir: "CheckTemplate"}},
		"SA1002": {{dir: "CheckTimeParse"}},
		"SA1003": {
			{dir: "CheckEncodingBinary"},
			{dir: "CheckEncodingBinary_go17", version: "1.7"},
			{dir: "CheckEncodingBinary_go18", version: "1.8"},
		},
		"SA1004": {{dir: "CheckTimeSleepConstant"}},
		"SA1005": {{dir: "CheckExec"}},
		"SA1006": {{dir: "CheckUnsafePrintf"}},
		"SA1007": {{dir: "CheckURLs"}},
		"SA1008": {{dir: "CheckCanonicalHeaderKey"}},
		"SA1010": {{dir: "checkStdlibUsageRegexpFindAll"}},
		"SA1011": {{dir: "checkStdlibUsageUTF8Cutset"}},
		"SA1012": {{dir: "checkStdlibUsageNilContext"}},
		"SA1013": {{dir: "checkStdlibUsageSeeker"}},
		"SA1014": {{dir: "CheckUnmarshalPointer"}},
		"SA1015": {
			{dir: "CheckLeakyTimeTick"},
			{dir: "CheckLeakyTimeTick-main"},
		},
		"SA1016": {{dir: "CheckUntrappableSignal"}},
		"SA1017": {{dir: "CheckUnbufferedSignalChan"}},
		"SA1018": {{dir: "CheckStringsReplaceZero"}},
		"SA1019": {
			{dir: "CheckDeprecated"},
			{dir: "CheckDeprecated_go14", version: "1.4"},
			{dir: "CheckDeprecated_go18", version: "1.8"},
		},
		"SA1020": {{dir: "CheckListenAddress"}},
		"SA1021": {{dir: "CheckBytesEqualIP"}},
		"SA1023": {{dir: "CheckWriterBufferModified"}},
		"SA1024": {{dir: "CheckNonUniqueCutset"}},
		"SA1025": {{dir: "CheckTimerResetReturnValue"}},
		"SA1026": {{dir: "CheckUnsupportedMarshal"}},
		"SA2000": {{dir: "CheckWaitgroupAdd"}},
		"SA2001": {{dir: "CheckEmptyCriticalSection"}},
		"SA2002": {{dir: "CheckConcurrentTesting"}},
		"SA2003": {{dir: "CheckDeferLock"}},
		"SA3000": {
			{dir: "CheckTestMainExit-1"},
			{dir: "CheckTestMainExit-2"},
			{dir: "CheckTestMainExit-3"},
			{dir: "CheckTestMainExit-4"},
			{dir: "CheckTestMainExit-5"},
		},
		"SA3001": {{dir: "CheckBenchmarkN"}},
		"SA4000": {{dir: "CheckLhsRhsIdentical"}},
		"SA4001": {{dir: "CheckIneffectiveCopy"}},
		"SA4002": {{dir: "CheckDiffSizeComparison"}},
		"SA4003": {{dir: "CheckExtremeComparison"}},
		"SA4004": {{dir: "CheckIneffectiveLoop"}},
		"SA4006": {{dir: "CheckUnreadVariableValues"}},
		"SA4008": {{dir: "CheckLoopCondition"}},
		"SA4009": {{dir: "CheckArgOverwritten"}},
		"SA4010": {{dir: "CheckIneffectiveAppend"}},
		"SA4011": {{dir: "CheckScopedBreak"}},
		"SA4012": {{dir: "CheckNaNComparison"}},
		"SA4013": {{dir: "CheckDoubleNegation"}},
		"SA4014": {{dir: "CheckRepeatedIfElse"}},
		"SA4015": {{dir: "CheckMathInt"}},
		"SA4016": {{dir: "CheckSillyBitwiseOps"}},
		"SA4017": {{dir: "CheckPureFunctions"}},
		"SA4018": {{dir: "CheckSelfAssignment"}},
		"SA4019": {{dir: "CheckDuplicateBuildConstraints"}},
		"SA4020": {{dir: "CheckUnreachableTypeCases"}},
		"SA4021": {{dir: "CheckSingleArgAppend"}},
		"SA5000": {{dir: "CheckNilMaps"}},
		"SA5001": {{dir: "CheckEarlyDefer"}},
		"SA5002": {{dir: "CheckInfiniteEmptyLoop"}},
		"SA5003": {{dir: "CheckDeferInInfiniteLoop"}},
		"SA5004": {{dir: "CheckLoopEmptyDefault"}},
		"SA5005": {{dir: "CheckCyclicFinalizer"}},
		"SA5007": {{dir: "CheckInfiniteRecursion"}},
		"SA5008": {{dir: "CheckStructTags"}},
		"SA5009": {{dir: "CheckPrintf"}},
		"SA6000": {{dir: "CheckRegexpMatchLoop"}},
		"SA6002": {{dir: "CheckSyncPoolValue"}},
		"SA6003": {{dir: "CheckRangeStringRunes"}},
		"SA6005": {{dir: "CheckToLowerToUpperComparison"}},
		"SA9001": {{dir: "CheckDubiousDeferInChannelRangeLoop"}},
		"SA9002": {{dir: "CheckNonOctalFileMode"}},
		"SA9003": {{dir: "CheckEmptyBranch"}},
		"SA9004": {{dir: "CheckMissingEnumTypesInDeclaration"}},
		"SA9005": {{dir: "CheckNoopMarshal"}},
	}

	for check, dirs := range checks {
		a := Analyzers[check]
		for _, dir := range dirs {
			if dir.version != "" {
				if err := a.Flags.Lookup("go").Value.Set(dir.version); err != nil {
					t.Fatal(err)
				}
			}
			analysistest.Run(t, analysistest.TestData(), a, dir.dir)
		}
	}
}
