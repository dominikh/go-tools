package sa9010

import (
	"fmt"
	"go/types"

	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/internal/passes/buildir"
	"honnef.co/go/tools/knowledge"

	"golang.org/x/exp/typeparams"
	"golang.org/x/tools/go/analysis"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA9010",
		Run:      run,
		Requires: []*analysis.Analyzer{buildir.Analyzer},
	},
	Doc: &lint.RawDocumentation{
		Title: "Conversion of uncomparable type to error interface",
		Text: `Converting a value of an uncomparable type to the error interface
can lead to runtime panics when error values are compared. Types containing
slices, maps, or function fields are not comparable.

For example:

    type MyError struct { Details []string }
    func (e MyError) Error() string { return "" }

    a, b := error(MyError{}), error(MyError{})
    fmt.Println(a == b) // panic: comparing uncomparable type

To avoid this, either use a pointer receiver or wrap the error.`,
		Since:    "2026.1",
		Severity: lint.SeverityWarning,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

func run(pass *analysis.Pass) (any, error) {
	irpkg := pass.ResultOf[buildir.Analyzer].(*buildir.IR)
	errorIface := knowledge.Interfaces["error"]

	for _, fn := range irpkg.SrcFuncs {
		for _, b := range fn.Blocks {
			for _, instr := range b.Instrs {
				makeIface, ok := instr.(*ir.MakeInterface)
				if !ok {
					continue
				}

				ifaceType, ok := makeIface.Type().Underlying().(*types.Interface)
				if !ok || !types.Identical(ifaceType, errorIface) {
					continue
				}

				srcType := makeIface.X.Type()

				if typeparams.IsTypeParam(srcType) {
					continue
				}

				if _, isPtr := srcType.Underlying().(*types.Pointer); isPtr {
					continue
				}

				if types.Comparable(srcType) {
					continue
				}

				report.Report(pass, makeIface,
					fmt.Sprintf("conversion of uncomparable type %s to error",
						types.TypeString(srcType, types.RelativeTo(pass.Pkg))))
			}
		}
	}
	return nil, nil
}
