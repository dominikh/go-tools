package qf1013

import (
	"go/ast"
	"go/token"
	"strings"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/edit"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const escapeCharacter = '\\'

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "QF1013",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	Doc: &lint.RawDocumentation{
		Title:    `Simplify string by using raw string literal`,
		Since:    "2024.8",
		Before:   `"a string with an escape quote: \""`,
		After:    "`a string with an escape quote: \"`",
		Severity: lint.SeverityHint,
	},
})

func run(pass *analysis.Pass) (any, error) {
	fn := func(node ast.Node) {
		lit, ok := node.(*ast.BasicLit)
		if !ok {
			return // not a basic lit
		}
		if lit.Kind != token.STRING {
			return // not a string
		}

		if strings.HasPrefix(lit.Value, "`") {
			// already a raw string
			return
		}
		val := lit.Value

		// in string literals, any character may appear except back quote.
		if strings.Contains(val, "`") {
			// so, we cannot transform it to a raw string if a back quote appears
			return
		}

		// this quickfix is intended to write simpler strings by using string literal
		// but it is limited to the following use cases:
		// - a quote is escaped
		// - a backslash is escaped
		// - everything else is ignored
		if !strings.Contains(val, string(escapeCharacter)) {
			// no blackslash in the string
			// nothing to do
			return
		}

		var cleanedVal string
		var escapeCharacterFound bool
		for _, c := range val[1 : len(val)-1] {
			if !escapeCharacterFound {
				escapeCharacterFound = (c == escapeCharacter)
				if !escapeCharacterFound {
					cleanedVal += string(c)
				}
				continue
			}

			// so the previous character was a backslash
			// we reset the flag for next character in the string
			escapeCharacterFound = false

			switch c {
			case escapeCharacter:
				// we have an escaped backslash
			case '"':
				// we have an escaped quote
			default:
				// currently unsupported
				return
			}
			cleanedVal += string(c)
		}

		msg := "Simplify string by using raw string literal"
		fix := edit.Fix(msg, edit.ReplaceWithNode(pass.Fset, node, &ast.BasicLit{
			Value: "`" + cleanedVal + "`",
		}))
		report.Report(pass, node, msg, report.Fixes(fix))
	}
	code.Preorder(pass, fn, (*ast.BasicLit)(nil))
	return nil, nil
}
