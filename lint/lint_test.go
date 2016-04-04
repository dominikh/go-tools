// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

package lint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestLine(t *testing.T) {
	tests := []struct {
		src    string
		offset int
		want   string
	}{
		{"single line file", 5, "single line file"},
		{"single line file with newline\n", 5, "single line file with newline\n"},
		{"first\nsecond\nthird\n", 2, "first\n"},
		{"first\nsecond\nthird\n", 9, "second\n"},
		{"first\nsecond\nthird\n", 14, "third\n"},
		{"first\nsecond\nthird with no newline", 16, "third with no newline"},
		{"first byte\n", 0, "first byte\n"},
	}
	for _, test := range tests {
		got := SrcLine([]byte(test.src), token.Position{Offset: test.offset})
		if got != test.want {
			t.Errorf("srcLine(%q, offset=%d) = %q, want %q", test.src, test.offset, got, test.want)
		}
	}
}

func TestExportedType(t *testing.T) {
	tests := []struct {
		typString string
		exp       bool
	}{
		{"int", true},
		{"string", false}, // references the shadowed builtin "string"
		{"T", true},
		{"t", false},
		{"*T", true},
		{"*t", false},
		{"map[int]complex128", true},
	}
	for _, test := range tests {
		src := `package foo; type T int; type t int; type string struct{}`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "foo.go", src, 0)
		if err != nil {
			t.Fatalf("Parsing %q: %v", src, err)
		}
		// use the package name as package path
		config := &types.Config{}
		pkg, err := config.Check(file.Name.Name, fset, []*ast.File{file}, nil)
		if err != nil {
			t.Fatalf("Type checking %q: %v", src, err)
		}
		tv, err := types.Eval(fset, pkg, token.NoPos, test.typString)
		if err != nil {
			t.Errorf("types.Eval(%q): %v", test.typString, err)
			continue
		}
		if got := ExportedType(tv.Type); got != test.exp {
			t.Errorf("exportedType(%v) = %t, want %t", tv.Type, got, test.exp)
		}
	}
}
