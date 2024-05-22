// Deprecated: deprecating external test packages is silly but possible.
package pkg_test

import (
	"testing"

	// External tests can import deprecated packages under test.
	pkg "example.com/CheckDeprecated"
)

// Deprecated: deprecating tests is silly but possible.
func TestFoo(t *testing.T) {
}

// This test isn't deprecated, to test that s.Field doesn't get flagged because it's from the package under test. If
// TestBar was itself deprecated, it could use any deprecated objects it wanted.
func TestBar(t *testing.T) {
	var s pkg.S
	// External tests can use deprecated objects from the package under test
	_ = s.Field
}
