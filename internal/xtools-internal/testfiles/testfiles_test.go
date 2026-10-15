// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testfiles_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/txtar"
	"honnef.co/go/tools/internal/xtools-internal/testenv"
	"honnef.co/go/tools/internal/xtools-internal/testfiles"
	"honnef.co/go/tools/internal/xtools-internal/versions"
)

func TestTestDir(t *testing.T) {
	testenv.NeedsGo1Point(t, 23)

	// Files are initially {go.mod.test,sub.test/sub.go.test}.
	fs := os.DirFS(filepath.Join(analysistest.TestData(), "versions"))
	tmpdir := testfiles.CopyToTmp(t, fs,
		"go.mod.test,go.mod",                // After: {go.mod,sub.test/sub.go.test}
		"sub.test/sub.go.test,sub.test/abc", // After: {go.mod,sub.test/abc}
		"sub.test,sub",                      // After: {go.mod,sub/abc}
		"sub/abc,sub/sub.go",                // After: {go.mod,sub/sub.go}
	)

	filever := &analysis.Analyzer{
		Name: "filever",
		Doc:  "reports file go versions",
		Run: func(pass *analysis.Pass) (any, error) {
			for _, file := range pass.Files {
				ver := versions.FileVersion(pass.TypesInfo, file)
				name := filepath.Base(pass.Fset.Position(file.Package).Filename)
				pass.Reportf(file.Package, "%s@%s", name, ver)
			}
			return nil, nil
		},
	}
	res := analysistest.Run(t, tmpdir, filever, "golang.org/fake/versions", "golang.org/fake/versions/sub")
	got := 0
	for _, r := range res {
		got += len(r.Diagnostics)
	}

	if want := 4; got != want {
		t.Errorf("Got %d diagnostics. wanted %d", got, want)
	}
}

func TestTestDirErrors(t *testing.T) {
	const input = `
-- one.txt --
one
`
	// Files are initially {go.mod.test,sub.test/sub.go.test}.
	fs, err := txtar.FS(txtar.Parse([]byte(input)))
	if err != nil {
		t.Fatal(err)
	}

	directive := "no comma to split on"
	intercept := &fatalIntercept{t, nil}
	func() {
		defer func() { // swallow panics from fatalIntercept.Fatal
			if r := recover(); r != intercept {
				panic(r)
			}
		}()
		testfiles.CopyToTmp(intercept, fs, directive)
	}()

	got := fmt.Sprint(intercept.fatalfs)
	want := `[rename directive "no comma to split on" does not contain delimiter ","]`
	if got != want {
		t.Errorf("CopyToTmp(%q) had the Fatal messages %q. wanted %q", directive, got, want)
	}
}

func TestLoadPackagesForcesModuleMode(t *testing.T) {
	t.Setenv("GO111MODULE", "off")

	const input = `
-- go.mod --
module example.com/test

go 1.23
-- main.go --
package test
`
	archive := txtar.Parse([]byte(input))
	pkgs := testfiles.LoadPackages(t, archive, ".")
	if len(pkgs) != 1 {
		t.Fatalf("LoadPackages returned %d packages, want 1", len(pkgs))
	}
	if got := pkgs[0].PkgPath; got != "example.com/test" {
		t.Fatalf("LoadPackages returned package path %q, want %q", got, "example.com/test")
	}
}

// helper for TestTestDirErrors
type fatalIntercept struct {
	testing.TB
	fatalfs []string
}

func (i *fatalIntercept) Fatalf(format string, args ...any) {
	i.fatalfs = append(i.fatalfs, fmt.Sprintf(format, args...))
	// Do not mark the test as failing, but fail early.
	panic(i)
}
