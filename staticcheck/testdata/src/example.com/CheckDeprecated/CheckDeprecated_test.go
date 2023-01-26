package pkg

import "testing"

// Deprecated: deprecating tests is silly but possible.
func TestFoo(t *testing.T) {
	var s S
	// Internal tests can use deprecated objects from the package they test.
	_ = s.Field
}

// This test isn't deprecated, to test that s.Field doesn't get flagged because it's from the package under test. If
// TestBar was itself deprecated, it could use any deprecated objects it wanted.
func TestBar(t *testing.T) {
	var s S
	// Internal tests can use deprecated objects from the package under test
	_ = s.Field
}
