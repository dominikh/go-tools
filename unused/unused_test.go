package unused

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/expect"
)

type expectation uint8

const (
	shouldBeUsed = iota
	shouldBeUnused
	shouldBeQuiet
)

func (exp expectation) String() string {
	switch exp {
	case shouldBeUsed:
		return "used"
	case shouldBeUnused:
		return "unused"
	case shouldBeQuiet:
		return "quiet"
	default:
		panic("unreachable")
	}
}

type key struct {
	ident string
	file  string
	line  int
}

func (k key) String() string {
	return fmt.Sprintf("%s:%d", k.file, k.line)
}

func relativePath(s string) string {
	// This is only used in a test, so we don't care about failures, or the cost of repeatedly calling os.Getwd
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	s, err = filepath.Rel(cwd, s)
	if err != nil {
		panic(err)
	}
	return s
}

func relativePosition(pos token.Position) string {
	s := pos.Filename
	if pos.IsValid() {
		if s != "" {
			// This is only used in a test, so we don't care about failures, or the cost of repeatedly calling os.Getwd
			cwd, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			s, err = filepath.Rel(cwd, s)
			if err != nil {
				panic(err)
			}
			s += ":"
		}
		s += fmt.Sprintf("%d", pos.Line)
		if pos.Column != 0 {
			s += fmt.Sprintf(":%d", pos.Column)
		}
	}
	if s == "" {
		s = "-"
	}
	return s
}

func check(t *testing.T, res *analysistest.Result) {
	want := map[key]expectation{}
	files := map[string]struct{}{}

	isTest := false
	for _, f := range res.Pass.Files {
		filename := res.Pass.Fset.Position(f.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			isTest = true
			break
		}
	}
	for _, f := range res.Pass.Files {
		filename := res.Pass.Fset.Position(f.Pos()).Filename
		if !strings.HasSuffix(filename, ".go") {
			continue
		}
		files[filename] = struct{}{}
		notes, err := expect.ExtractGo(res.Pass.Fset, f)
		if err != nil {
			t.Fatal(err)
		}
		for _, note := range notes {
			posn := res.Pass.Fset.PositionFor(note.Pos, false)
			switch note.Name {
			case "quiet":
				if len(note.Args) != 1 {
					t.Fatalf("malformed directive at %s", posn)
				}

				if !isTest {
					want[key{note.Args[0].(string), posn.Filename, posn.Line}] = expectation(shouldBeQuiet)
				}
			case "quiet_test":
				if len(note.Args) != 1 {
					t.Fatalf("malformed directive at %s", posn)
				}

				if isTest {
					want[key{note.Args[0].(string), posn.Filename, posn.Line}] = expectation(shouldBeQuiet)
				}
			case "used":
				if len(note.Args) != 2 {
					t.Fatalf("malformed directive at %s", posn)
				}

				if !isTest {
					var e expectation
					if note.Args[1].(bool) {
						e = shouldBeUsed
					} else {
						e = shouldBeUnused
					}
					want[key{note.Args[0].(string), posn.Filename, posn.Line}] = e
				}
			case "used_test":
				if len(note.Args) != 2 {
					t.Fatalf("malformed directive at %s", posn)
				}

				if isTest {
					var e expectation
					if note.Args[1].(bool) {
						e = shouldBeUsed
					} else {
						e = shouldBeUnused
					}
					want[key{note.Args[0].(string), posn.Filename, posn.Line}] = expectation(e)
				}
			}
		}
	}

	checkObjs := func(objs []Object, state expectation) {
		for _, obj := range objs {
			// if t, ok := obj.Type().(*types.Named); ok && t.TypeArgs().Len() != 0 {
			// 	continue
			// }
			posn := obj.Position
			if _, ok := files[posn.Filename]; !ok {
				continue
			}

			// This key isn't great. Because of generics, multiple objects (instantiations of a generic type) exist at
			// the same location. This only works because we ignore instantiations, but may lead to confusing test failures.
			k := key{obj.ShortName, posn.Filename, posn.Line}
			exp, ok := want[k]
			if !ok {
				t.Errorf("object at %s (%s) shouldn't exist but is %s (tests = %t)", relativePosition(posn), obj.ShortName, state, isTest)
				continue
			}
			if false {
				// Sometimes useful during debugging, but too noisy to have enabled for all test failures
				t.Logf("%s handled by %q", k, obj)
			}
			delete(want, k)
			if state != exp {
				t.Errorf("object at %s (%s) should be %s but is %s (tests = %t)", relativePosition(posn), obj.ShortName, exp, state, isTest)
			}
		}
	}
	ures := res.Result.(Result)
	checkObjs(ures.Used, shouldBeUsed)
	checkObjs(ures.Unused, shouldBeUnused)
	checkObjs(ures.Quiet, shouldBeQuiet)

	for key, e := range want {
		exp := e.String()
		t.Errorf("object at %s:%d should be %s but wasn't seen", relativePath(key.file), key.line, exp)
	}
}

func TestAll(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join(analysistest.TestData(), "src", "example.com", "*"))
	if err != nil {
		t.Fatal(err)
	}
	for i, dir := range dirs {
		dirs[i] = filepath.Join("example.com", filepath.Base(dir))
	}

	results := analysistest.Run(t, analysistest.TestData(), Analyzer.Analyzer, dirs...)
	for _, res := range results {
		check(t, res)
	}
}
