package simple

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAll(t *testing.T) {
	checks := map[string][]struct {
		dir     string
		version string
	}{
		"S1000": {{dir: "single-case-select"}},
		"S1001": {{dir: "copy"}},
		"S1002": {{dir: "bool-cmp"}},
		"S1003": {{dir: "contains"}},
		"S1004": {{dir: "compare"}},
		"S1005": {
			{dir: "LintBlankOK"},
			{dir: "receive-blank"},
			{dir: "range_go13", version: "1.3"},
			{dir: "range_go14", version: "1.4"},
		},
		"S1006": {
			{dir: "for-true"},
			{dir: "generated"},
		},
		"S1007": {{dir: "regexp-raw"}},
		"S1008": {{dir: "if-return"}},
		"S1009": {{dir: "nil-len"}},
		"S1010": {{dir: "slicing"}},
		"S1011": {{dir: "loop-append"}},
		"S1012": {{dir: "time-since"}},
		"S1016": {
			{dir: "convert"},
			{dir: "convert_go17", version: "1.7"},
			{dir: "convert_go18", version: "1.8"},
		},
		"S1017": {{dir: "trim"}},
		"S1018": {{dir: "LintLoopSlide"}},
		"S1019": {{dir: "LintMakeLenCap"}},
		"S1020": {{dir: "LintAssertNotNil"}},
		"S1021": {{dir: "LintDeclareAssign"}},
		"S1023": {
			{dir: "LintRedundantBreak"},
			{dir: "LintRedundantReturn"},
		},
		"S1024": {
			{dir: "LimeTimeUntil_go17", version: "1.7"},
			{dir: "LimeTimeUntil_go18", version: "1.8"},
		},
		"S1025": {{dir: "LintRedundantSprintf"}},
		"S1028": {{dir: "LintErrorsNewSprintf"}},
		"S1029": {{dir: "LintRangeStringRunes"}},
		"S1030": {{dir: "LintBytesBufferConversions"}},
		"S1031": {{dir: "LintNilCheckAroundRange"}},
		"S1032": {{dir: "LintSortHelpers"}},
		"S1033": {{dir: "LintGuardedDelete"}},
		"S1034": {{dir: "LintSimplifyTypeSwitch"}},
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
