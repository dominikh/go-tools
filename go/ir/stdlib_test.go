// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//lint:file-ignore SA1019 go/ssa's test suite is built around the deprecated go/loader. We'll leave fixing that to upstream.

// Incomplete source tree on Android.

//go:build !android
// +build !android

package ir_test

// This file runs the IR builder in sanity-checking mode on all
// packages beneath $GOROOT and prints some summary information.
//
// Run with "go test -cpu=8 to" set GOMAXPROCS.

import (
	"testing"

	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/go/ir/irutil"

	"golang.org/x/tools/go/packages"
)

func TestStdlib(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode; too slow (golang.org/issue/14113)")
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes,
	}
	pkgs, err := packages.Load(cfg, "std")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	for _, pkg := range pkgs {
		if len(pkg.Errors) != 0 {
			t.Fatalf("Load failed: %v", pkg.Errors[0])
		}
	}

	var mode ir.BuilderMode
	mode |= ir.SanityCheckFunctions
	mode |= ir.GlobalDebug
	prog, _ := irutil.Packages(pkgs, mode, nil)
	prog.Build()
}
