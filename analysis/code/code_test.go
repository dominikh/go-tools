package code

import "testing"

var constraintsFromNameTests = []struct {
	in  string
	out string
}{
	{"foo.go", ""},
	{"foo_windows.go", "windows"},
	{"foo_unix.go", ""},
	{"foo_windows_amd64.go", "windows && amd64"},
	{"foo_amd64.go", "amd64"},
	{"foo_windows_nonsense.go", ""},
	{"foo_nonsense_amd64.go", "amd64"},
	{"foo_nonsense_windows.go", "windows"},
	{"foo_nonsense_windows_amd64.go", "amd64"},
	{"foo_windows_test.go", "windows"},
	{"linux.go", ""},
	{"linux_amd64.go", "amd64"},
	{"amd64_linux.go", "linux"},
	{"amd64.go", ""},
}

func TestConstraintsFromName(t *testing.T) {
	for _, tc := range constraintsFromNameTests {
		expr := constraintsFromName(tc.in)
		var out string
		if expr != nil {
			out = expr.String()
		}
		if out != tc.out {
			t.Errorf("constraintsFromName(%q) == %q, expected %q", tc.in, out, tc.out)
		}
	}
}

func FuzzConstraintsFromName(f *testing.F) {
	for _, tc := range constraintsFromNameTests {
		f.Add(tc.in)
	}

	f.Fuzz(func(t *testing.T, name string) {
		constraintsFromName(name)
	})
}
