package sa4001

import (
	"go/ast"
	"regexp"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA4001",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	Doc: &lint.RawDocumentation{
		Title: `\'&*x\' gets simplified to \'x\', it does not copy \'x\'`,
		Text: `Combining address-taking and pointer dereferencing does not create a copy
of the pointed-to value. In particular, \'&*p\' still points to the same
object as \'p\'. To make a copy, assign \'*p\' to a new variable and take
that variable's address. This is a shallow copy: any pointers, slices, or
maps in the value still refer to the same underlying data.`,
		Since:    "2017.1",
		Severity: lint.SeverityWarning,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var (
	// cgo produces code like fn(&*_Cvar_kSomeCallbacks) which we don't
	// want to flag.
	cgoIdent               = regexp.MustCompile(`^_C(func|var)_.+$`)
	checkIneffectiveCopyQ1 = pattern.MustParse(`(UnaryExpr "&" (StarExpr obj))`)
	checkIneffectiveCopyQ2 = pattern.MustParse(`(StarExpr (UnaryExpr "&" _))`)
)

func run(pass *analysis.Pass) (any, error) {
	fn := func(node ast.Node) {
		if m, ok := code.Match(pass, checkIneffectiveCopyQ1, node); ok {
			if ident, ok := m.State["obj"].(*ast.Ident); !ok || !cgoIdent.MatchString(ident.Name) {
				report.Report(pass, node, "&*x will be simplified to x. It will not copy x.")
			}
		} else if _, ok := code.Match(pass, checkIneffectiveCopyQ2, node); ok {
			report.Report(pass, node, "*&x will be simplified to x. It will not copy x.")
		}
	}
	code.Preorder(pass, fn, (*ast.UnaryExpr)(nil), (*ast.StarExpr)(nil))
	return nil, nil
}
