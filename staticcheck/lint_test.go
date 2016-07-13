package staticcheck

import (
	"testing"

	"honnef.co/go/lint/testutil"
)

func TestAll(t *testing.T) {
	testutil.TestAll(t, Funcs, "")
	testutil.TestAll(t, DubiousFuncs, "dubious")
}
