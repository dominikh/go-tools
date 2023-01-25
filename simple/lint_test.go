package simple

import (
	"testing"

	"honnef.co/go/tools/analysis/lint/testutil"
)

func TestAll(t *testing.T) {
	checks := map[string][]testutil.Test{
		"S1000": {{Dir: "example.com/CheckSingleCaseSelect"}},
		"S1001": {{Dir: "example.com/CheckLoopCopy"}},
		"S1002": {{Dir: "example.com/CheckIfBoolCmp"}},
		"S1003": {{Dir: "example.com/CheckStringsContains"}},
		"S1004": {{Dir: "example.com/CheckBytesCompare"}},
		"S1005": {
			{Dir: "example.com/CheckUnnecessaryBlank"},
			{Dir: "example.com/CheckUnnecessaryBlank_go13", Version: "1.3"},
			{Dir: "example.com/CheckUnnecessaryBlank_go14", Version: "1.4"},
		},
		"S1006": {{Dir: "example.com/CheckForTrue"}},
		"S1007": {{Dir: "example.com/CheckRegexpRaw"}},
		"S1008": {{Dir: "example.com/CheckIfReturn"}},
		"S1009": {{Dir: "example.com/CheckRedundantNilCheckWithLen"}},
		"S1010": {{Dir: "example.com/CheckSlicing"}},
		"S1011": {{Dir: "example.com/CheckLoopAppend"}},
		"S1012": {{Dir: "example.com/CheckTimeSince"}},
		"S1016": {
			{Dir: "example.com/CheckSimplerStructConversion"},
			{Dir: "example.com/CheckSimplerStructConversion_go17", Version: "1.7"},
			{Dir: "example.com/CheckSimplerStructConversion_go18", Version: "1.8"},
		},
		"S1017": {{Dir: "example.com/CheckTrim"}},
		"S1018": {{Dir: "example.com/CheckLoopSlide"}},
		"S1019": {{Dir: "example.com/CheckMakeLenCap"}},
		"S1020": {{Dir: "example.com/CheckAssertNotNil"}},
		"S1021": {{Dir: "example.com/CheckDeclareAssign"}},
		"S1023": {{Dir: "example.com/CheckRedundantBreak"}, {Dir: "example.com/CheckRedundantReturn"}},
		"S1024": {{Dir: "example.com/CheckTimeUntil_go17", Version: "1.7"}, {Dir: "example.com/CheckTimeUntil_go18", Version: "1.8"}},
		"S1025": {{Dir: "example.com/CheckRedundantSprintf"}},
		"S1028": {{Dir: "example.com/CheckErrorsNewSprintf"}},
		"S1029": {{Dir: "example.com/CheckRangeStringRunes"}},
		"S1030": {{Dir: "example.com/CheckBytesBufferConversions"}},
		"S1031": {{Dir: "example.com/CheckNilCheckAroundRange"}},
		"S1032": {{Dir: "example.com/CheckSortHelpers"}},
		"S1033": {{Dir: "example.com/CheckGuardedDelete"}},
		"S1034": {{Dir: "example.com/CheckSimplifyTypeSwitch"}},
		"S1035": {{Dir: "example.com/CheckRedundantCanonicalHeaderKey"}},
		"S1036": {{Dir: "example.com/CheckUnnecessaryGuard"}},
		"S1037": {{Dir: "example.com/CheckElaborateSleep"}},
		"S1038": {{Dir: "example.com/CheckPrintSprintf"}},
		"S1039": {{Dir: "example.com/CheckSprintLiteral"}},
		"S1040": {{Dir: "example.com/CheckSameTypeTypeAssertion"}},
	}

	testutil.Run(t, Analyzers, checks)
}
