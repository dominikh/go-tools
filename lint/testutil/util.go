package testutil

import (
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

type Test struct {
	Dir     string
	Version string
}

func Run(t *testing.T, analyzers map[string]*analysis.Analyzer, tests map[string][]Test) {
	for _, a := range analyzers {
		a := a
		t.Run(a.Name, func(t *testing.T) {
			t.Parallel()
			tt, ok := tests[a.Name]
			if !ok {
				t.Fatalf("no tests for analyzer %s", a.Name)
			}
			for _, test := range tt {
				if test.Version != "" {
					if err := a.Flags.Lookup("go").Value.Set(test.Version); err != nil {
						t.Fatal(err)
					}
				}
				analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), a, test.Dir)
			}
		})
	}
}
