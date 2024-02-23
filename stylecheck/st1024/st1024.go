package st1024

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"

	"honnef.co/go/tools/analysis/facts/generated"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "ST1024",
		Run:      run,
		Requires: []*analysis.Analyzer{generated.Analyzer},
	},
	Doc: &lint.Documentation{
		Title: "Package aliases should not be redundant",
		Text: `An alias should be a non-trivial renaming of what it aliases.

For example, there's no benefit from using "bar" as an alias for a package
named "bar" whose package path is "example.com/foo/bar". We can just use
its name.`,
		MergeIf: lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

func run(pass *analysis.Pass) (interface{}, error) {
	// To avoid loading the same packages repeatedly, we construct a map of the
	// package path to the package pointer outside of the loops.
	pathToPackage := make(map[string]*packages.Package)

	for _, f := range pass.Files {
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return nil, fmt.Errorf("unquoting the package path %q: %w", imp.Path.Value, err)
			}

			var pkgName string
			pkg, ok := pathToPackage[path]
			if !ok {
				pkgsLoaded, err := packages.Load(&packages.Config{Mode: packages.NeedName}, path)
				if err != nil || len(pkgsLoaded) != 1 {
					return nil, fmt.Errorf("loading packages: %w", err)
				}
				pkg = pkgsLoaded[0]
				pathToPackage[path] = pkg
			}
			pkgName = pkg.Name

			// Is there a package alias? Is it different from the package name?
			if imp.Name == nil || imp.Name.Name != pkgName {
				continue
			}

			// We compare against the last component of the package as it's
			// stylistically reasonable to use a package alias when the package name
			// and the last component of its path differ. For example, the package
			// name of github.com/google/gofuzz is "fuzz"; explicitly importing it as
			// "fuzz" is reasonable.
			lastPkgComponent := path
			if idx := strings.LastIndexByte(lastPkgComponent, '/'); idx != -1 {
				lastPkgComponent = lastPkgComponent[idx+1:]
			}
			if pkgName != lastPkgComponent {
				continue
			}

			opts := []report.Option{report.FilterGenerated()}
			report.Report(pass, imp, fmt.Sprintf("package %q is imported with a redundant alias", path), opts...)
		}
	}

	return nil, nil
}
