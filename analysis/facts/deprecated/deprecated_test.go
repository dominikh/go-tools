package deprecated

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestDeprecated(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "example.com/Deprecated")
}
