package simple

import (
	"testing"

	"honnef.co/go/tools/lint/testutil"
)

func TestAll(t *testing.T) {
	checks := map[string][]testutil.Test{
		"S1000": {{Dir: "single-case-select"}},
		"S1001": {{Dir: "copy"}},
		"S1002": {{Dir: "bool-cmp"}},
		"S1003": {{Dir: "contains"}},
		"S1004": {{Dir: "compare"}},
		"S1005": {{Dir: "LintBlankOK"}, {Dir: "receive-blank"}, {Dir: "range_go13", Version: "1.3"}, {Dir: "range_go14", Version: "1.4"}},
		"S1006": {{Dir: "for-true"}, {Dir: "generated"}},
		"S1007": {{Dir: "regexp-raw"}},
		"S1008": {{Dir: "if-return"}},
		"S1009": {{Dir: "nil-len"}},
		"S1010": {{Dir: "slicing"}},
		"S1011": {{Dir: "loop-append"}},
		"S1012": {{Dir: "time-since"}},
		"S1016": {{Dir: "convert"}, {Dir: "convert_go17", Version: "1.7"}, {Dir: "convert_go18", Version: "1.8"}},
		"S1017": {{Dir: "trim"}},
		"S1018": {{Dir: "LintLoopSlide"}},
		"S1019": {{Dir: "LintMakeLenCap"}},
		"S1020": {{Dir: "LintAssertNotNil"}},
		"S1021": {{Dir: "LintDeclareAssign"}},
		"S1023": {{Dir: "LintRedundantBreak"}, {Dir: "LintRedundantReturn"}},
		"S1024": {{Dir: "LimeTimeUntil_go17", Version: "1.7"}, {Dir: "LimeTimeUntil_go18", Version: "1.8"}},
		"S1025": {{Dir: "LintRedundantSprintf"}},
		"S1028": {{Dir: "LintErrorsNewSprintf"}},
		"S1029": {{Dir: "LintRangeStringRunes"}},
		"S1030": {{Dir: "LintBytesBufferConversions"}},
		"S1031": {{Dir: "LintNilCheckAroundRange"}},
		"S1032": {{Dir: "LintSortHelpers"}},
		"S1033": {{Dir: "LintGuardedDelete"}},
		"S1034": {{Dir: "LintSimplifyTypeSwitch"}},
		"S1035": {{Dir: "LintRedundantCanonicalHeaderKey"}},
		"S1036": {{Dir: "LintUnnecessaryGuard"}},
		"S1037": {{Dir: "LintElaborateSleep"}},
	}

	testutil.Run(t, Analyzers, checks)
}
