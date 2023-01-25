package purity

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestPurity(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "example.com/Purity")
}
