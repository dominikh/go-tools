package sa1015

import (
	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/go/ir/irutil"
	"honnef.co/go/tools/internal/passes/buildir"

	"golang.org/x/tools/go/analysis"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA1015",
		Run:      run,
		Requires: []*analysis.Analyzer{buildir.Analyzer},
	},
	Doc: &lint.Documentation{
		Title:    `Using \'time.Tick\' in a way that will leak. Consider using \'time.NewTicker\', and only use \'time.Tick\' in tests, commands and endless functions`,
		Since:    "2017.1",
		Severity: lint.SeverityWarning,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

func run(pass *analysis.Pass) (interface{}, error) {
	for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
		if code.IsMainLike(pass) || code.IsInTest(pass, fn) {
			continue
		}
		for _, block := range fn.Blocks {
			for _, ins := range block.Instrs {
				call, ok := ins.(*ir.Call)
				if !ok || !irutil.IsCallTo(call.Common(), "time.Tick") {
					continue
				}
				if !irutil.Terminates(call.Parent()) {
					continue
				}
				report.Report(pass, call, "using time.Tick leaks the underlying ticker, consider using it only in endless functions, tests and the main package, and use time.NewTicker here")
			}
		}
	}
	return nil, nil
}
