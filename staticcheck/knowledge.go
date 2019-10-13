package staticcheck

import (
	"reflect"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/internal/passes/buildir"
	"honnef.co/go/tools/ir"
	"honnef.co/go/tools/staticcheck/vrp"
)

var valueRangesAnalyzer = &analysis.Analyzer{
	Name: "vrp",
	Doc:  "calculate value ranges of functions",
	Run: func(pass *analysis.Pass) (interface{}, error) {
		m := map[*ir.Function]vrp.Ranges{}
		for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
			vr := vrp.BuildGraph(fn).Solve()
			m[fn] = vr
		}
		return m, nil
	},
	Requires:   []*analysis.Analyzer{buildir.Analyzer},
	ResultType: reflect.TypeOf(map[*ir.Function]vrp.Ranges{}),
}
