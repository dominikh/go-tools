package pattern

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	inputs := []string{
		`(Binding "name" _)`,
		`(Binding "name" _:[])`,
		`(Binding "name" _:_:[])`,
	}

	p := Parser{}
	for _, input := range inputs {
		if _, err := p.Parse(input); err != nil {
			t.Errorf("failed to parse %q: %s", input, err)
		}
	}
}

func FuzzParse(f *testing.F) {
	var files []*ast.File
	fset := token.NewFileSet()

	// Ideally we'd check against as much source code as possible, but that's fairly slow, on the order of 500ms per
	// pattern when checking against the whole standard library.
	//
	// We pick the runtime package in the hopes that it contains the most diverse, and weird, code.
	filepath.Walk(runtime.GOROOT()+"/src/runtime", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// XXX error handling
			panic(err)
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := goparser.ParseFile(fset, path, nil, goparser.SkipObjectResolution)
		if err != nil {
			return nil
		}
		files = append(files, f)
		return nil
	})

	parse := func(in string, allowTypeInfo bool) (Pattern, bool) {
		p := Parser{
			AllowTypeInfo: allowTypeInfo,
		}
		pat, err := p.Parse(string(in))
		if err != nil {
			if strings.Contains(err.Error(), "internal error") {
				panic(err)
			}
			return Pattern{}, false
		}
		return pat, true
	}

	f.Fuzz(func(t *testing.T, in []byte) {
		defer func() {
			if err := recover(); err != nil {
				str := fmt.Sprint(err)
				if strings.Contains(str, "binding already created:") {
					// This is an invalid pattern, not a real failure
				} else {
					// Re-panic the original panic
					panic(err)
				}
			}
		}()
		// Parse twice, once with AllowTypeInfo set to true to exercise the parser, and once with it set to false so we
		// can actually use it in Match, as we don't have type information available.

		pat, ok := parse(string(in), true)
		if !ok {
			return
		}
		// Make sure we can turn it back into a string
		_ = pat.Root.String()

		pat, ok = parse(string(in), false)
		if !ok {
			return
		}
		// Make sure we can turn it back into a string
		_ = pat.Root.String()

		// Don't check patterns with too many relevant nodes; it's too expensive
		if len(pat.Relevant) < 20 {
			// Make sure trying to match nodes doesn't panic
			for _, f := range files {
				ast.Inspect(f, func(node ast.Node) bool {
					rt := reflect.TypeOf(node)
					// We'd prefer calling Match on all nodes, not just those the pattern deems relevant, to find more bugs.
					// However, doing so has a 10x cost in execution time.
					if _, ok := pat.Relevant[rt]; ok {
						Match(pat, node)
					}
					return true
				})
			}
		}
	})
}
