package simple

import (
	"testing"

	"honnef.co/go/tools/analysis/lint/testutil"
)

func TestAll(t *testing.T) {
	checks := map[string][]testutil.Test{
		"S1000": {{Dir: "CheckSingleCaseSelect"}},
		"S1001": {{Dir: "CheckLoopCopy"}},
		"S1002": {{Dir: "CheckIfBoolCmp"}},
		"S1003": {{Dir: "CheckStringsContains"}},
		"S1004": {{Dir: "CheckBytesCompare"}},
		"S1005": {
			{Dir: "CheckUnnecessaryBlank"},
			{Dir: "CheckUnnecessaryBlank_go13", Version: "1.3"},
			{Dir: "CheckUnnecessaryBlank_go14", Version: "1.4"},
		},
		"S1006": {{Dir: "CheckForTrue"}},
		"S1007": {{Dir: "CheckRegexpRaw"}},
		"S1008": {{Dir: "CheckIfReturn"}},
		"S1009": {{Dir: "CheckRedundantNilCheckWithLen"}},
		"S1010": {{Dir: "CheckSlicing"}},
		"S1011": {{Dir: "CheckLoopAppend"}},
		"S1012": {{Dir: "CheckTimeSince"}},
		"S1016": {
			{Dir: "CheckSimplerStructConversion"},
			{Dir: "CheckSimplerStructConversion_go17", Version: "1.7"},
			{Dir: "CheckSimplerStructConversion_go18", Version: "1.8"},
		},
		"S1017": {{Dir: "CheckTrim"}},
		"S1018": {{Dir: "CheckLoopSlide"}},
		"S1019": {{Dir: "CheckMakeLenCap"}},
		"S1020": {{Dir: "CheckAssertNotNil"}},
		"S1021": {{Dir: "CheckDeclareAssign"}},
		"S1023": {{Dir: "CheckRedundantBreak"}, {Dir: "CheckRedundantReturn"}},
		"S1024": {{Dir: "CheckTimeUntil_go17", Version: "1.7"}, {Dir: "CheckTimeUntil_go18", Version: "1.8"}},
		"S1025": {{Dir: "CheckRedundantSprintf"}},
		"S1028": {{Dir: "CheckErrorsNewSprintf"}},
		"S1029": {{Dir: "CheckRangeStringRunes"}},
		"S1030": {{Dir: "CheckBytesBufferConversions"}},
		"S1031": {{Dir: "CheckNilCheckAroundRange"}},
		"S1032": {{Dir: "CheckSortHelpers"}},
		"S1033": {{Dir: "CheckGuardedDelete"}},
		"S1034": {{Dir: "CheckSimplifyTypeSwitch"}},
		"S1035": {{Dir: "CheckRedundantCanonicalHeaderKey"}},
		"S1036": {{Dir: "CheckUnnecessaryGuard"}},
		"S1037": {{Dir: "CheckElaborateSleep"}},
		"S1038": {{Dir: "CheckPrintSprintf"}},
		"S1039": {{Dir: "CheckSprintLiteral"}},
		"S1040": {{Dir: "CheckSameTypeTypeAssertion"}},
	}

	testutil.Run(t, Analyzers, checks)
}
