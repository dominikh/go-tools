package lint

import (
	"go/ast"
	"go/token"
	"reflect"

	"golang.org/x/tools/go/analysis"
)

var IsGeneratedAnalyzer = &analysis.Analyzer{
	Name: "isgenerated",
	Doc:  "annotate file names that have been code generated",
	Run: func(pass *analysis.Pass) (interface{}, error) {
		m := map[string]bool{}
		for _, f := range pass.Files {
			path := pass.Fset.PositionFor(f.Pos(), false).Filename
			m[path] = isGenerated(path)
		}
		return m, nil
	},
	RunDespiteErrors: true,
	ResultType:       reflect.TypeOf(map[string]bool{}),
}

var TokenFileAnalyzer = &analysis.Analyzer{
	Name: "tokenfileanalyzer",
	Doc:  "creates a mapping of *token.File to *ast.File",
	Run: func(pass *analysis.Pass) (interface{}, error) {
		m := map[*token.File]*ast.File{}
		for _, af := range pass.Files {
			tf := pass.Fset.File(af.Pos())
			m[tf] = af
		}
		return m, nil
	},
	RunDespiteErrors: true,
	ResultType:       reflect.TypeOf(map[*token.File]*ast.File{}),
}
