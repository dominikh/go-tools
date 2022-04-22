package unused

import (
	"fmt"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/exp/typeparams"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/expect"
)

type expectation bool

const (
	shouldBeUsed   = true
	shouldBeUnused = false
)

func (exp expectation) String() string {
	switch exp {
	case shouldBeUsed:
		return "used"
	case shouldBeUnused:
		return "unused"
	default:
		panic("unreachable")
	}
}

type key struct {
	file string
	line int
}

func (k key) String() string {
	return fmt.Sprintf("%s:%d", k.file, k.line)
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
			case "used":
				if !isTest {
					want[key{posn.Filename, posn.Line}] = expectation(note.Args[0].(bool))
				}
			case "used_test":
				if isTest {
					want[key{posn.Filename, posn.Line}] = expectation(note.Args[0].(bool))
				}
			}
		}
	}

	checkObjs := func(objs []types.Object, state expectation) {
		for _, obj := range objs {
			if obj, ok := obj.(*types.TypeName); ok {
				if _, ok := obj.Type().(*typeparams.TypeParam); ok {
					// we don't care about type parameters
					continue
				}
			}
			if t, ok := obj.Type().(*types.Named); ok && typeparams.NamedTypeArgs(t).Len() != 0 {
				continue
			}
			posn := res.Pass.Fset.Position(obj.Pos())
			if _, ok := files[posn.Filename]; !ok {
				continue
			}

			// This key isn't great. Because of generics, multiple objects (instantiations of a generic type) exist at
			// the same location. This only works because we ignore instantiations, but may lead to confusing test failures.
			k := key{posn.Filename, posn.Line}
			exp, ok := want[k]
			if !ok {
				t.Errorf("unexpected %s object %q at %s", state, obj, posn)
				continue
			}
			if false {
				// Sometimes useful during debugging, but too noisy to have enabled for all test failures
				t.Logf("%s handled by %q", k, obj)
			}
			delete(want, k)
			if state != exp {
				t.Errorf("object at %s should be %s but is %s", posn, exp, state)
			}
		}
	}
	ures := res.Result.(Result)
	checkObjs(ures.Used, shouldBeUsed)
	checkObjs(ures.Unused, shouldBeUnused)

	for key, b := range want {
		var exp string
		if b {
			exp = "used"
		} else {
			exp = "unused"
		}
		t.Errorf("did not see expected %s object %s:%d", exp, key.file, key.line)
	}
}

func TestAll(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join(analysistest.TestData(), "src", "*"))
	if err != nil {
		t.Fatal(err)
	}
	for i, dir := range dirs {
		dirs[i] = filepath.Base(dir)
	}

	results := analysistest.Run(t, analysistest.TestData(), Analyzer.Analyzer, dirs...)
	for _, res := range results {
		check(t, res)
	}
}
