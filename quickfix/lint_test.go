package quickfix

import (
	"testing"

	"honnef.co/go/tools/lint/testutil"
)

func TestAll(t *testing.T) {
	checks := map[string][]testutil.Test{
		"QF1000": {{Dir: "CheckStringsIndexByte"}},
	}

	testutil.Run(t, Analyzers, checks)
}
