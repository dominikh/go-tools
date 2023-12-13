package sa9009

import (
	"fmt"
	"regexp"

	"golang.org/x/tools/go/analysis"

	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA9009",
		Run:      run,
		Requires: []*analysis.Analyzer{},
	},
	Doc: &lint.RawDocumentation{
		Title: "Ineffectual Go compiler directive",
		Text: `
A potential Go compiler directive was found, but is ineffectual as it begins
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
