package lint_test

import (
	"testing"

	. "honnef.co/go/tools/lint"
	"honnef.co/go/tools/lint/testutil"
)

type testChecker struct{}

func (testChecker) Name() string       { return "stylecheck" }
func (testChecker) Prefix() string     { return "TEST" }
func (testChecker) Init(prog *Program) {}

func (testChecker) Funcs() map[string]Func {
	return map[string]Func{
		"TEST1000": testLint,
	}
}

func testLint(j *Job) {
	// Flag all functions
	for _, fn := range j.Program.InitialFunctions {
		if fn.Synthetic == "" {
			j.Errorf(fn, "This is a test problem")
		}
	}
}

func TestAll(t *testing.T) {
	c := testChecker{}
	testutil.TestAll(t, c, "")
}
