package staticcheck

import (
	"testing"

	"honnef.co/go/lint/testutil"
)

func TestAll(t *testing.T) {
	c := NewChecker()
	testutil.TestAll(t, c, "")
}
