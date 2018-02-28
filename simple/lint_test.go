package simple

import (
	"testing"

	"github.com/cabify/go-tools/lint/testutil"
)

func TestAll(t *testing.T) {
	testutil.TestAll(t, NewChecker(), "")
}
