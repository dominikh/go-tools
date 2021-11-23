//go:build go1.18
// +build go1.18

package lintcmd

import (
	"testing"
)

var buildConfigTests = []struct {
	in      string
	name    string
	envs    []string
	flags   []string
	invalid bool
}{
	{
		`some_name: ENV1=foo ENV_2=bar ENV3="foo bar baz" ENV4=foo"bar" -flag1 -flag2= -flag3=value -flag4="some value" -flag5=some" value "test "-flag6=1"`,
		"some_name",
		[]string{"ENV1=foo", "ENV_2=bar", "ENV3=foo bar baz", "ENV4=foobar"},
		[]string{"-flag1", "-flag2=", "-flag3=value", "-flag4=some value", "-flag5=some value test", "-flag6=1"},
		false,
	},
	{
		`some_name: ENV1=foo -tags bar baz=meow`,
		"some_name",
		[]string{"ENV1=foo"},
		[]string{"-tags", "bar", "baz=meow"},
		false,
	},
	{
		"some_name:",
		"some_name",
		nil,
		nil,
		false,
	},
	{
		"some name:",
		"",
		nil,
		nil,
		true,
	},
	{
		"",
		"",
		nil,
		nil,
		true,
	},
}

func FuzzParseBuildConfig(f *testing.F) {
	equal := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	for _, tt := range buildConfigTests {
		f.Add(tt.in)

		name, envs, flags, err := parseBuildConfig(tt.in)
		if err != nil {
			if tt.invalid {
				continue
			}
			f.Fatalf("input %q failed to parse: %s", tt.in, err)
		}
		if tt.invalid {
			f.Fatalf("expected input %q to fail but it didn't", tt.in)
		}

		if name != tt.name {
			f.Fatalf("got name %q, want %q", name, tt.name)
		}
		if !equal(envs, tt.envs) {
			f.Fatalf("got environment %#v, want %#v", envs, tt.envs)
		}
		if !equal(flags, tt.flags) {
			f.Fatalf("got flags %#v, want %#v", flags, tt.flags)
		}
	}

	f.Fuzz(func(t *testing.T, in string) {
		parseBuildConfig(in)
	})
}
