package sa1025

import (
	"go/types"

	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/go/ir/irutil"
	"honnef.co/go/tools/internal/passes/buildir"

	"golang.org/x/tools/go/analysis"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA1025",
		Run:      run,
		Requires: []*analysis.Analyzer{buildir.Analyzer},
	},
	Doc: &lint.RawDocumentation{
		Title: `It is not possible to use \'(*time.Timer).Reset\''s return value correctly`,
		Text: `When using pre-Go 1.23 timer semantics, draining a timer's channel based on
\'Reset\''s return value is racy: the new timer may expire before the channel
is drained, causing the new notification to be consumed instead of the old
one. Stop the timer and, if necessary, drain its channel before calling
\'Reset\'. This must be coordinated with any other receivers of the channel.

With Go 1.23 timer semantics, \'Reset\' guarantees that subsequent receives
will not observe stale values from the previous timer settings. There is
no need to drain the channel when resetting such a timer.`,
		Since:    "2019.1",
		Severity: lint.SeverityWarning,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

func run(pass *analysis.Pass) (any, error) {
	for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
		for _, block := range fn.Blocks {
			for _, ins := range block.Instrs {
				call, ok := ins.(*ir.Call)
				if !ok {
					continue
				}
				if !irutil.IsCallTo(call.Common(), "(*time.Timer).Reset") {
					continue
				}
				refs := call.Referrers()
				if refs == nil {
					continue
				}
				for _, ref := range *refs {
					ifstmt, ok := ref.(*ir.If)
					if !ok {
						continue
					}

					found := false
					for _, succ := range ifstmt.Block().Succs {
						if len(succ.Preds) != 1 {
							// Merge point, not a branch in the
							// syntactical sense.

							// FIXME(dh): this is broken for if
							// statements a la "if x || y"
							continue
						}
						irutil.Walk(succ, func(b *ir.BasicBlock) bool {
							if !succ.Dominates(b) {
								// We've reached the end of the branch
								return false
							}
							for _, ins := range b.Instrs {
								// TODO(dh): we should check that we're receiving from the
								// channel of a time.Timer to further reduce false
								// positives. Not a key priority, considering the rarity
								// of Reset and the tiny likeliness of a false positive
								//
								// We intentionally don't handle aliases here, because
								// we're only interested in time.Timer.C.
								if ins, ok := ins.(*ir.Recv); ok && types.TypeString(ins.Chan.Type(), nil) == "<-chan time.Time" {
									found = true
									return false
								}
							}
							return true
						})
					}

					if found {
						report.Report(pass, call, "it is not possible to use Reset's return value correctly, as there is a race condition between draining the channel and the new timer expiring")
					}
				}
			}
		}
	}
	return nil, nil
}
