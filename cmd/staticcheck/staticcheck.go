// staticcheck analyses Go code and makes it better.
package main // import "honnef.co/go/tools/cmd/staticcheck"

import (
	"os"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/lint"
	"honnef.co/go/tools/lint/lintutil"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
	"honnef.co/go/tools/unused"
)

func main() {
	fs := lintutil.FlagSet("staticcheck")
	fs.Parse(os.Args[1:])

	var cs []*analysis.Analyzer
	for _, v := range simple.Analyzers {
		cs = append(cs, v)
	}
	for _, v := range staticcheck.Analyzers {
		cs = append(cs, v)
	}
	for _, v := range stylecheck.Analyzers {
		cs = append(cs, v)
	}

	cums := []lint.CumulativeChecker{unused.NewChecker()}

	lintutil.ProcessFlagSet(cs, cums, fs)
}
