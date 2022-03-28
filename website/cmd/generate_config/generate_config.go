package main

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"

	"github.com/BurntSushi/toml"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/config"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
	"honnef.co/go/tools/unused"
)

func main() {
	cfg := config.DefaultConfig

	checks := []string{"all"}
	do := func(analyzers ...*lint.Analyzer) {
		for _, a := range analyzers {
			if a.Doc.NonDefault {
				// Use backticks to quote the check name so TOML doesn't escape them
				checks = append(checks, fmt.Sprintf("-{{< check `%s` >}}", a.Analyzer.Name))
			}
		}
	}
	do(simple.Analyzers...)
	do(staticcheck.Analyzers...)
	do(stylecheck.Analyzers...)
	do(unused.Analyzer)
	do(quickfix.Analyzers...)

	sort.Slice(checks[1:], func(i, j int) bool {
		return checks[i+1] < checks[j+1]
	})

	cfg.Checks = checks

	buf := bytes.Buffer{}
	toml.NewEncoder(&buf).Encode(cfg)

	r := regexp.MustCompile(`(?m)^[a-z_]+`)
	out := r.ReplaceAllString(buf.String(), "{{< option `$0` >}}")

	fmt.Println("---")
	fmt.Println("headless: true")
	fmt.Println("---")
	fmt.Println("```toml")
	fmt.Print(out)
	fmt.Println("```")
}
