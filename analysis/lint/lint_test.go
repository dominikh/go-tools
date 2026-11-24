package lint

import (
	"testing"

	"golang.org/x/tools/go/analysis"
)

var testDocs = []RawDocumentation{
	{Title: `Invalid regular expression`, Since: "2017.1", Severity: SeverityError},
	{
		Title: `Don't use \'fmt.Sprintf("%s", x)\' unnecessarily`,
		Text: `In many instances, there are easier and more efficient ways
of getting a value's string representation. Use \"Stringer\" or
\'strconv\' instead.`,
		Before:     "\n  x := fmt.Sprintf(\"%s\", s)\n",
		After:      "\n  x := s\n",
		Since:      "2019.1",
		NonDefault: true,
		Options:    []string{"dot_import_whitelist", "http_status_code_whitelist"},
		Severity:   SeverityWarning,
		MergeIf:    MergeIfAll,
	},
	{Title: "Unreleased check"},
}

func TestInitializeAnalyzerDoc(t *testing.T) {
	for _, raw := range testDocs {
		a := InitializeAnalyzer(&Analyzer{Analyzer: &analysis.Analyzer{Name: "TEST"}, Doc: &raw})
		if want := raw.Compile().String(); a.Analyzer.Doc != want {
			t.Errorf("Doc = %q, want %q", a.Analyzer.Doc, want)
		}
	}
}

func TestCompileMarkdown(t *testing.T) {
	doc := testDocs[1].Compile()
	if want := "Don't use `fmt.Sprintf(\"%s\", x)` unnecessarily"; doc.TitleMarkdown != want {
		t.Errorf("TitleMarkdown = %q, want %q", doc.TitleMarkdown, want)
	}
	if want := "Don't use fmt.Sprintf(\"%s\", x) unnecessarily"; doc.Title != want {
		t.Errorf("Title = %q, want %q", doc.Title, want)
	}
}

func BenchmarkInitializeAnalyzer(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for _, raw := range testDocs {
			InitializeAnalyzer(&Analyzer{Analyzer: &analysis.Analyzer{Name: "TEST"}, Doc: &raw})
		}
	}
}
