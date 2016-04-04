// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// Package lintutil provides helpers for writing linter command lines.
package lintutil

import (
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"honnef.co/go/lint"
)

func usage(name string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", name)
		fmt.Fprintf(os.Stderr, "\t%s [flags] # runs on package in current directory\n", name)
		fmt.Fprintf(os.Stderr, "\t%s [flags] package\n", name)
		fmt.Fprintf(os.Stderr, "\t%s [flags] directory\n", name)
		fmt.Fprintf(os.Stderr, "\t%s [flags] files... # must be a single package\n", name)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
}

type runner struct {
	funcs         []lint.Func
	minConfidence float64
}

func ProcessArgs(name string, funcs []lint.Func, args []string) {
	flags := flag.FlagSet{
		Usage: usage(name),
	}
	var minConfidence = flags.Float64("min_confidence", 0.8, "minimum confidence of a problem to print it")
	flags.Parse(args)

	runner := runner{funcs, *minConfidence}
	switch flags.NArg() {
	case 0:
		runner.lintDir(".")
	case 1:
		arg := flags.Arg(0)
		if strings.HasSuffix(arg, "/...") && isDir(arg[:len(arg)-4]) {
			for _, dirname := range allPackagesInFS(arg) {
				runner.lintDir(dirname)
			}
		} else if isDir(arg) {
			runner.lintDir(arg)
		} else if exists(arg) {
			runner.lintFiles(arg)
		} else {
			for _, pkgname := range importPaths([]string{arg}) {
				runner.lintPackage(pkgname)
			}
		}
	default:
		runner.lintFiles(flags.Args()...)
	}
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func (runner runner) lintFiles(filenames ...string) {
	files := make(map[string][]byte)
	for _, filename := range filenames {
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		files[filename] = src
	}

	l := &lint.Linter{
		Funcs: runner.funcs,
	}
	ps, err := l.LintFiles(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	for _, p := range ps {
		if p.Confidence >= runner.minConfidence {
			fmt.Printf("%v: %s\n", p.Position, p.Text)
		}
	}
}

func (runner runner) lintDir(dirname string) {
	pkg, err := build.ImportDir(dirname, 0)
	runner.lintImportedPackage(pkg, err)
}

func (runner runner) lintPackage(pkgname string) {
	pkg, err := build.Import(pkgname, ".", 0)
	runner.lintImportedPackage(pkg, err)
}

func (runner runner) lintImportedPackage(pkg *build.Package, err error) {
	if err != nil {
		if _, nogo := err.(*build.NoGoError); nogo {
			// Don't complain if the failure is due to no Go source files.
			return
		}
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var files []string
	xtest := pkg.XTestGoFiles
	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	files = append(files, pkg.TestGoFiles...)
	if pkg.Dir != "." {
		for i, f := range files {
			files[i] = filepath.Join(pkg.Dir, f)
		}
		for i, f := range xtest {
			xtest[i] = filepath.Join(pkg.Dir, f)
		}
	}
	runner.lintFiles(xtest...)
	runner.lintFiles(files...)
}
