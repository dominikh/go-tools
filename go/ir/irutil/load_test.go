// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package irutil_test

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path"
	"strings"
	"testing"

	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/go/ir/irutil"
	"honnef.co/go/tools/internal/xtools-internal/packagestest"
	"honnef.co/go/tools/internal/xtools-internal/testenv"

	"golang.org/x/tools/go/packages"
)

const hello = `package main

import "fmt"

func main() {
	fmt.Println("Hello, world")
}
`

func TestBuildPackage(t *testing.T) {
	testenv.NeedsGoBuild(t) // for importer.Default()

	// There is a more substantial test of BuildPackage and the
	// IR program it builds in ../ir/builder_test.go.

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hello.go", hello, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}

	for _, mode := range []ir.BuilderMode{
		ir.SanityCheckFunctions,
		ir.InstantiateGenerics | ir.SanityCheckFunctions,
	} {
		pkg := types.NewPackage("hello", "")
		irpkg, _, err := irutil.BuildPackage(&types.Config{Importer: importer.Default()}, fset, pkg, []*ast.File{f}, mode)
		if err != nil {
			t.Fatal(err)
		}
		if pkg.Name() != "main" {
			t.Errorf("pkg.Name() = %s, want main", pkg.Name())
		}
		if irpkg.Func("main") == nil {
			irpkg.WriteTo(os.Stderr)
			t.Errorf("irpkg has no main function")
		}

	}
}

func TestPackages(t *testing.T) {
	testenv.NeedsGoPackages(t)

	cfg := &packages.Config{Mode: packages.LoadSyntax}
	initial, err := packages.Load(cfg, "bytes")
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(initial) > 0 {
		t.Fatal("there were errors")
	}

	for _, mode := range []ir.BuilderMode{
		ir.SanityCheckFunctions,
		ir.SanityCheckFunctions | ir.InstantiateGenerics,
	} {
		prog, pkgs := irutil.Packages(initial, mode)
		bytesNewBuffer := pkgs[0].Func("NewBuffer")
		bytesNewBuffer.Pkg.Build()

		// We'll dump the IR of bytes.NewBuffer because it is small and stable.
		out := new(bytes.Buffer)
		bytesNewBuffer.WriteTo(out)

		// For determinism, sanitize the location.
		location := prog.Fset.Position(bytesNewBuffer.Pos()).String()
		got := strings.Replace(out.String(), location, "$GOROOT/src/bytes/buffer.go:1", -1)

		want := `
# Name: bytes.NewBuffer
# Package: bytes
# Location: $GOROOT/src/bytes/buffer.go:1
func NewBuffer(buf []byte) *Buffer:
0:                                                                entry P:0 S:0
        t1 = new Buffer (complit)                                       *Buffer
        t2 = {buf: buf, off: 0:int, lastRead: 0:readOp}                  Buffer
        *t1 = t2
        return t1

`[1:]
		if got != want {
			t.Errorf("bytes.NewBuffer IR = <<%s>>, want <<%s>>", got, want)
		}
	}
}

func TestBuildPackage_MissingImport(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "bad.go", `package bad; import "missing"`, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}

	pkg := types.NewPackage("bad", "")
	irpkg, _, err := irutil.BuildPackage(new(types.Config), fset, pkg, []*ast.File{f}, 0)
	if err == nil || irpkg != nil {
		t.Fatal("BuildPackage succeeded unexpectedly")
	}
}

func TestIssue28106(t *testing.T) {
	testenv.NeedsGoPackages(t)

	// In go1.10, go/packages loads all packages from source, not
	// export data, but does not type check function bodies of
	// imported packages. This test ensures that we do not attempt
	// to run the IR builder on functions without type information.
	cfg := &packages.Config{Mode: packages.LoadSyntax}
	pkgs, err := packages.Load(cfg, "runtime")
	if err != nil {
		t.Fatal(err)
	}
	prog, _ := irutil.Packages(pkgs, 0)
	prog.Build() // no crash
}

func TestIssue53604(t *testing.T) {
	// Tests that variable initializers are not added to init() when syntax
	// is not present but types.Info is available.
	//
	// Packages x, y, z are loaded with mode `packages.LoadSyntax`.
	// Package x imports y, and y imports z.
	// Packages are built using irutil.Packages() with x and z as roots.
	// This setup creates y using CreatePackage(pkg, files, info, ...)
	// where len(files) == 0 but info != nil.
	//
	// Tests that globals from y are not initialized.
	e := packagestest.Export(t, packagestest.Modules, []packagestest.Module{
		{
			Name: "golang.org/fake",
			Files: map[string]any{
				"x/x.go": `package x; import "golang.org/fake/y"; var V = y.F()`,
				"y/y.go": `package y; import "golang.org/fake/z"; var F = func () *int { return &z.Z } `,
				"z/z.go": `package z; var Z int`,
			},
		},
	})
	defer e.Cleanup()

	// Load x and z as entry packages using packages.LoadSyntax
	e.Config.Mode = packages.LoadSyntax
	pkgs, err := packages.Load(e.Config, path.Join(e.Temp(), "fake/x"), path.Join(e.Temp(), "fake/z"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pkgs {
		if len(p.Errors) > 0 {
			t.Fatalf("%v", p.Errors)
		}
	}

	prog, _ := irutil.Packages(pkgs, ir.BuilderMode(0))
	prog.Build()

	// y does not initialize F.
	y := prog.ImportedPackage("golang.org/fake/y")
	if y == nil {
		t.Fatal("Failed to load intermediate package y")
	}
	yinit := y.Members["init"].(*ir.Function)
	for _, bb := range yinit.Blocks {
		for _, i := range bb.Instrs {
			if store, ok := i.(*ir.Store); ok && store.Addr == y.Var("F") {
				t.Errorf("y.init() stores to F %v", store)
			}
		}
	}
}
