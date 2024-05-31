package lintcmd

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestFilterAnalyzerNames(t *testing.T) {
	analyzers := []string{
		"S1000", "S1001",
		"SA1000", "SA1001",
		"SA2000", "SA2001",
		"ST1000", "ST1001",
		"U1000",
	}
	all := make(map[string]bool)
	for _, a := range analyzers {
		all[a] = true
	}
	allMinus := func(minus ...string) map[string]bool {
		m := make(map[string]bool)
		for k := range all {
			m[k] = !slices.Contains(minus, k)
		}
		return m
	}

	tests := []struct {
		in      []string
		want    map[string]bool
		wantErr string
	}{
		{[]string{"all"}, all, ""},
		{[]string{"All"}, all, ""},
		{[]string{"*"}, all, ""},

		{[]string{"S*"}, map[string]bool{"S1000": true, "S1001": true}, ""},
		{[]string{"SA1*"}, map[string]bool{"SA1000": true, "SA1001": true}, ""},
		{[]string{"SA2*"}, map[string]bool{"SA2000": true, "SA2001": true}, ""},
		{[]string{"SA*"}, map[string]bool{"SA1000": true, "SA1001": true, "SA2000": true, "SA2001": true}, ""},

		{[]string{"S1000", "st1000"}, map[string]bool{"S1000": true, "ST1000": true}, ""},

		{[]string{"SA9*"}, nil, "matched no checks"},
		{[]string{"S*", "SA9*"}, nil, "matched no checks"},
		{[]string{"SA9*", "all"}, nil, "matched no checks"},

		{[]string{"S9999"}, nil, "unknown check"},
		{[]string{"check"}, nil, "unknown check"},
		{[]string{`!@#'"`}, nil, "unknown check"},

		{[]string{"all", "-S1000"}, allMinus("S1000"), ""},
		{[]string{"all", "-SA1*"}, allMinus("SA1000", "SA1001"), ""},
		{[]string{"-S1000", "all"}, all, ""},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			have, haveErr := filterAnalyzerNames(analyzers, tt.in)
			if !errorContains(haveErr, tt.wantErr) {
				t.Fatalf("wrong error:\nhave: %s\nwant: %s", haveErr, tt.wantErr)
			}
			if !reflect.DeepEqual(have, tt.want) {
				t.Fatalf("\nhave: %v\nwant: %v", have, tt.want)
			}
		})
	}
}

func errorContains(have error, want string) bool {
	if have == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(have.Error(), want)
}
