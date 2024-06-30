package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

type Output struct {
	Checks     map[string]*lint.Documentation
	ByCategory map[string][]string
}

func category(check string) string {
	idx := strings.IndexAny(check, "0123456789")
	return check[:idx+1]
}

func main() {
	output := Output{
		Checks:     map[string]*lint.Documentation{},
		ByCategory: map[string][]string{},
	}

	groups := [][]*lint.Analyzer{
		staticcheck.Analyzers,
		simple.Analyzers,
		stylecheck.Analyzers,
		quickfix.Analyzers,
	}
	for _, group := range groups {
		for _, a := range group {
			doc := a.Doc.Compile()
			doc.Text = convertText(doc.Text)
			doc.TextMarkdown = convertText(doc.TextMarkdown)
			output.Checks[a.Analyzer.Name] = doc
			g := output.ByCategory[category(a.Analyzer.Name)]
			output.ByCategory[category(a.Analyzer.Name)] = append(g, a.Analyzer.Name)
		}
	}

	for _, v := range output.ByCategory {
		sort.Strings(v)
	}

	out, err := json.MarshalIndent(output, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func moreCodeFollows(lines []string) bool {
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "    ") {
			return true
		} else {
			return false
		}
	}
	return false
}

var alpha = regexp.MustCompile(`^[a-zA-Z ]+$`)

func convertText(text string) string {
	var buf bytes.Buffer
	lines := strings.Split(text, "\n")

	inCode := false
	empties := 0
	for i, line := range lines {
		if inCode {
			if !moreCodeFollows(lines[i:]) {
				if inCode {
					fmt.Fprintln(&buf, "```")
					inCode = false
				}
			}
		}

		prevEmpties := empties
		if line == "" && !inCode {
			empties++
		} else {
			empties = 0
		}

		if line == "" {
			fmt.Fprintln(&buf)
			continue
		}

		if strings.HasPrefix(line, "    ") {
			line = line[4:]
			if !inCode {
				fmt.Fprintln(&buf, "```go")
				inCode = true
			}
		}

		onlyAlpha := alpha.MatchString(line)
		out := line
		if !inCode && prevEmpties >= 2 && onlyAlpha {
			fmt.Fprintf(&buf, "#### %s\n", out)
		} else {
			fmt.Fprint(&buf, out)
			fmt.Fprintln(&buf)
		}
	}
	if inCode {
		fmt.Fprintln(&buf, "```")
	}

	return strings.TrimSpace(buf.String())
}
