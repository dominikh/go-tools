package sa9009

import (
	"fmt"
	"regexp"

	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"

	"golang.org/x/tools/go/analysis"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA9009",
		Run:      run,
		Requires: []*analysis.Analyzer{},
	},
	Doc: &lint.Documentation{
		Title: "ineffectual go compiler directive",
		Text: `
A go compiler directive was found, but is ineffectual as it begins
with whitespace.`,
		Since:    "Unreleased",
		Severity: lint.SeverityWarning,
	},
})

var Analyzer = SCAnalyzer.Analyzer

func run(pass *analysis.Pass) (any, error) {
	re := regexp.MustCompile("^[ \t]*(//|/\\*)[ \t]+go:")
	for _, f := range pass.Files {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if re.FindStringIndex(c.Text) == nil {
					continue
				}
				report.Report(pass, c,
					fmt.Sprintf("ineffectual compiler directive: %q", c.Text))
			}
		}
	}
	return nil, nil
}
