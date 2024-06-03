//go:build ignore

package main

import (
	"bytes"
	"go/format"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

var tmpl = `
package {{.lname}}

import (
	"honnef.co/go/tools/analysis/lint"

	"golang.org/x/tools/go/analysis"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name: "{{.name}}",
		Run: run,
		Requires: []*analysis.Analyzer{},
	},
	Doc: &lint.RawDocumentation{
		Title: "",
		Text: {{.emptyRaw}},
		{{- if .quickfix }}
		Before: {{.emptyRaw}},
		After: {{.emptyRaw}},
		{{- end }}
		Since: "Unreleased",
		Severity: lint.SeverityWarning,
	},
})

var Analyzer = SCAnalyzer.Analyzer

func run(pass *analysis.Pass) (any, error) {
	return nil, nil
}
`

func main() {
	log.SetFlags(0)

	var t template.Template
	if _, err := t.Parse(tmpl); err != nil {
		log.Fatalln("couldn't parse template:", err)
	}

	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <new check's name>", os.Args[0])
	}

	name := os.Args[1]
	checkRe := regexp.MustCompile(`^([A-Za-z]+)\d{4}$`)
	parts := checkRe.FindStringSubmatch(name)
	if parts == nil {
		log.Fatalf("invalid check name %q", name)
	}

	var catDir string
	prefix := strings.ToUpper(parts[1])
	switch prefix {
	case "SA":
		catDir = "staticcheck"
	case "S":
		catDir = "simple"
	case "ST":
		catDir = "stylecheck"
	case "QF":
		catDir = "quickfix"
	default:
		log.Fatalf("unknown check prefix %q", prefix)
	}

	lname := strings.ToLower(name)
	dir := filepath.Join(catDir, lname)
	dst := filepath.Join(dir, lname+".go")

	mkdirp(dir)

	buf := bytes.NewBuffer(nil)
	vars := map[string]any{
		"name":     name,
		"lname":    lname,
		"emptyRaw": "``",
		"quickfix": prefix == "QF",
	}

	if err := t.Execute(buf, vars); err != nil {
		log.Fatalf("couldn't generate %s: %s", dst, err)
	}

	b, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("couldn't gofmt %s: %s", dst, err)
	}

	writeFile(dst, b)

	testdata := filepath.Join(dir, "testdata", "src", "example.com", "pkg")
	mkdirp(testdata)
	writeFile(filepath.Join(testdata, "pkg.go"), []byte("package pkg\n"))

	out, err := exec.Command("go", "generate", "./...").CombinedOutput()
	if err != nil {
		log.Printf("could not run 'go generate ./...': %s", err)
		log.Println("Output:")
		log.Fatalln(string(out))
	}

	flags := []string{
		"add",
		"--intent-to-add",
		"--verbose",

		filepath.Join(dir, lname+"_test.go"),
		filepath.Join(testdata, "pkg.go"),
		dst,
	}
	cmd := exec.Command("git", flags...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalln("could not run 'git add':", err)
	}
}

func writeFile(path string, data []byte) {
	if err := os.WriteFile(path, data, 0677); err != nil {
		log.Fatalf("couldn't write %s: %s", path, err)
	}
}

func mkdirp(path string) {
	if err := os.MkdirAll(path, 0777); err != nil {
		log.Fatalf("couldn't create directory %s: %s", path, err)
	}
}
