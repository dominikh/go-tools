package sa4026

import (
	"fmt"
	"go/ast"
	"go/types"
	"strconv"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/edit"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA4026",
		Run:      run,
		Requires: code.RequiredAnalyzers,
	},
	Doc: &lint.RawDocumentation{
		Title: "Go constants cannot express negative zero",
		Text: `In IEEE 754 floating point math, zero has a sign and can be positive
or negative. This can be useful in certain numerical code.

Go constants, however, cannot express negative zero. This means that
the literals \'-0.0\' and \'0.0\' have the same ideal value (zero) and
will both represent positive zero at runtime.

To explicitly and reliably create a negative zero, you can use the
\'math.Copysign\' function: \'math.Copysign(0, -1)\'.`,
		Since:    "2021.1",
		Severity: lint.SeverityWarning,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var negativeZeroFloatQ = pattern.MustParse(`
	(Or
		(UnaryExpr
			"-"
			(BasicLit "FLOAT" "0.0"))

		(UnaryExpr
			"-"
			(CallExpr conv@(Object (Or "float32" "float64")) lit@(Or (BasicLit "INT" "0") (BasicLit "FLOAT" "0.0"))))

		(CallExpr
			conv@(Object (Or "float32" "float64"))
			(UnaryExpr "-" lit@(BasicLit "INT" "0"))))`)

func run(pass *analysis.Pass) (any, error) {
	mathQualifier := func(node ast.Node) string {
		for _, file := range pass.Files {
			if node.Pos() < file.Pos() || node.Pos() > file.End() {
				continue
			}
			for _, spec := range file.Imports {
				path, err := strconv.Unquote(spec.Path.Value)
				if err != nil || path != "math" {
					continue
				}
				var pkgName *types.PkgName
				if spec.Name != nil {
					pkgName, _ = pass.TypesInfo.Defs[spec.Name].(*types.PkgName)
				} else {
					pkgName, _ = pass.TypesInfo.Implicits[spec].(*types.PkgName)
				}
				if pkgName != nil && pkgName.Name() != "." && pkgName.Name() != "_" {
					scope := pass.TypesInfo.Scopes[file].Innermost(node.Pos())
					if scope != nil {
						_, obj := scope.LookupParent(pkgName.Name(), node.Pos())
						if obj == pkgName {
							return pkgName.Name()
						}
					}
				}
			}
		}
		return ""
	}
	for node, m := range code.Matches(pass, negativeZeroFloatQ) {
		mathName := mathQualifier(node)
		if conv, ok := m.State["conv"].(*types.TypeName); ok {
			var replacement string
			// TODO(dh): how does this handle type aliases?
			if conv.Name() == "float32" {
				replacement = fmt.Sprintf(`float32(%s.Copysign(0, -1))`, mathName)
			} else {
				replacement = fmt.Sprintf(`%s.Copysign(0, -1)`, mathName)
			}
			var opts []report.Option
			if mathName != "" {
				opts = append(opts, report.Fixes(edit.Fix("Use math.Copysign to create negative zero", edit.ReplaceWithString(node, replacement))))
			}
			report.Report(pass, node,
				fmt.Sprintf("in Go, the floating-point expression '%s' is the same as '%s(%s)', it does not produce a negative zero",
					report.Render(pass, node),
					conv.Name(),
					report.Render(pass, m.State["lit"])),
				opts...)
		} else {
			var opts []report.Option
			if mathName != "" {
				opts = append(opts, report.Fixes(edit.Fix("Use math.Copysign to create negative zero", edit.ReplaceWithString(node, fmt.Sprintf(`%s.Copysign(0, -1)`, mathName)))))
			}
			report.Report(pass, node,
				"in Go, the floating-point literal '-0.0' is the same as '0.0', it does not produce a negative zero",
				opts...)
		}
	}
	return nil, nil
}
