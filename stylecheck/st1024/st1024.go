package st1024

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/internal/passes/buildir"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "ST1024",
		Run:      run,
		Requires: []*analysis.Analyzer{buildir.Analyzer},
	},
	Doc: &lint.RawDocumentation{
		Title: "Avoid named error return values named 'err'",
		Text: `Named return values called 'err' are discouraged because they can lead
to confusion and subtle mistakes.

In most Go code, 'err' is used for short-lived local variables. Naming a
return parameter 'err' can induce cognitive load.

For example:

	func fn() (err error) {
		if err := doSomething(); err != nil { // shadows the return
			return err
		}
		defer func() {qqq
			// This 'err' is the named return, not the one above
			if err != nil {
				log.Println("deferred error:", err)
			}
		}()
		return nil
	}

Using a distinct name for the named return makes the return value’s
role explicit and avoids confusion:

	func fn() (retErr error) {
		if err := doSomething(); err != nil {
			return err
		}
		defer func() {
			if retErr != nil {
				log.Println("deferred error:", retErr)
			}
		}()
		return nil
	}

This check is opt-in and not enabled by default.`,
		Since:      "2025.2",
		NonDefault: true,
		MergeIf:    lint.MergeIfAll,
	},
})

func run(pass *analysis.Pass) (interface{}, error) {
	errorType := types.Universe.Lookup("error").Type()

fnLoop:
	for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
		sig := fn.Type().(*types.Signature)
		rets := sig.Results()
		if rets == nil {
			continue
		}
		for i := range rets.Len() {
			if r := rets.At(i); types.Unalias(r.Type()) == errorType && r.Name() == "err" {
				report.Report(pass, rets.At(i), "named error return should not be named 'err'", report.ShortRange())
				continue fnLoop
			}
		}
	}
	return nil, nil
}
