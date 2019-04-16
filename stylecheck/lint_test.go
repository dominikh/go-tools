package stylecheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAll(t *testing.T) {
	checks := map[string][]struct {
		dir     string
		version string
	}{
		"ST1000": {
			{dir: "CheckPackageComment-1"},
			{dir: "CheckPackageComment-2"},
		},
		"ST1001": {{dir: "CheckDotImports"}},
		"ST1003": {
			{dir: "CheckNames"},
			{dir: "CheckNames_generated"},
		},
		"ST1005": {{dir: "CheckErrorStrings"}},
		"ST1006": {{dir: "CheckReceiverNames"}},
		"ST1008": {{dir: "CheckErrorReturn"}},
		"ST1011": {{dir: "CheckTimeNames"}},
		"ST1012": {{dir: "CheckErrorVarNames"}},
		"ST1013": {{dir: "CheckHTTPStatusCodes"}},
		"ST1015": {{dir: "CheckDefaultCaseOrder"}},
		"ST1016": {{dir: "CheckReceiverNamesIdentical"}},
		"ST1017": {{dir: "CheckYodaConditions"}},
		"ST1018": {{dir: "CheckInvisibleCharacters"}},
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
