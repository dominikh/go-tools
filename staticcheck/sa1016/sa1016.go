package sa1016

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
		Name:     "SA1016",
		Run:      run,
		Requires: code.RequiredAnalyzers,
	},
	Doc: &lint.RawDocumentation{
		Title: `Trapping a signal that cannot be trapped`,
		Text: `Not all signals can be intercepted by a process. Specifically, on
UNIX-like systems, the \'syscall.SIGKILL\' and \'syscall.SIGSTOP\' signals are
never passed to the process, but instead handled directly by the
kernel. It is therefore pointless to try and handle these signals.`,
		Since:    "2017.1",
		Severity: lint.SeverityWarning,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var query = pattern.MustParse(`
	(CallExpr
		(Symbol
			(Or
				"os/signal.Ignore"
				"os/signal.Notify"
				"os/signal.Reset"))
		_)`)

func run(pass *analysis.Pass) (any, error) {
	isSignal := func(pass *analysis.Pass, expr ast.Expr, name string) bool {
		if expr, ok := expr.(*ast.SelectorExpr); ok {
			return code.SelectorName(pass, expr) == name
		} else {
			return false
		}
	}

	for node := range code.Matches(pass, query) {
		call := node.(*ast.CallExpr)
		hasSigterm := false
		for _, arg := range call.Args {
			if conv, ok := arg.(*ast.CallExpr); ok && isSignal(pass, conv.Fun, "os.Signal") {
				arg = conv.Args[0]
			}

			if isSignal(pass, arg, "syscall.SIGTERM") {
				hasSigterm = true
				break
			}

		}
		for i, arg := range call.Args {
			if conv, ok := arg.(*ast.CallExpr); ok && isSignal(pass, conv.Fun, "os.Signal") {
				arg = conv.Args[0]
			}

			if isSignal(pass, arg, "os.Kill") || isSignal(pass, arg, "syscall.SIGKILL") {
				var fixes []analysis.SuggestedFix
				if !hasSigterm {
					if syscallAlias := importAlias(pass, call, "syscall"); syscallAlias != "" {
						nargs := make([]ast.Expr, len(call.Args))
						for j, a := range call.Args {
							if i == j {
								nargs[j] = edit.Selector(syscallAlias, "SIGTERM")
							} else {
								nargs[j] = a
							}
						}
						ncall := *call
						ncall.Args = nargs
						fixes = append(fixes, edit.Fix(fmt.Sprintf("Use syscall.SIGTERM instead of %s", report.Render(pass, arg)), edit.ReplaceWithNode(pass.Fset, call, &ncall)))
					}
				}
				nargs := make([]ast.Expr, 0, len(call.Args))
				for j, a := range call.Args {
					if i == j {
						continue
					}
					nargs = append(nargs, a)
				}
				ncall := *call
				ncall.Args = nargs
				fixes = append(fixes, edit.Fix(fmt.Sprintf("Remove %s from list of arguments", report.Render(pass, arg)), edit.ReplaceWithNode(pass.Fset, call, &ncall)))
				report.Report(pass, arg, fmt.Sprintf("%s cannot be trapped (did you mean syscall.SIGTERM?)", report.Render(pass, arg)), report.Fixes(fixes...))
			}
			if isSignal(pass, arg, "syscall.SIGSTOP") {
				nargs := make([]ast.Expr, 0, len(call.Args)-1)
				for j, a := range call.Args {
					if i == j {
						continue
					}
					nargs = append(nargs, a)
				}
				ncall := *call
				ncall.Args = nargs
				report.Report(pass, arg, "syscall.SIGSTOP cannot be trapped", report.Fixes(edit.Fix("Remove syscall.SIGSTOP from list of arguments", edit.ReplaceWithNode(pass.Fset, call, &ncall))))
			}
		}
	}
	return nil, nil
}

func importAlias(pass *analysis.Pass, node ast.Node, importPath string) string {
	for _, spec := range code.File(pass, node).Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil || p != importPath {
			continue
		}
		var pkgName *types.PkgName
		if spec.Name == nil {
			pkgName, _ = pass.TypesInfo.Implicits[spec].(*types.PkgName)
		} else {
			pkgName, _ = pass.TypesInfo.Defs[spec.Name].(*types.PkgName)
		}
		if pkgName == nil || pkgName.Name() == "." || pkgName.Name() == "_" {
			return ""
		}
		scope := pass.TypesInfo.Scopes[code.File(pass, node)].Innermost(node.Pos())
		if scope != nil {
			_, obj := scope.LookupParent(pkgName.Name(), node.Pos())
			if obj == pkgName {
				return pkgName.Name()
			}
		}
		return ""
	}
	return ""
}
