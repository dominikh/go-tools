// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// Package lintutil provides helpers for writing linter command lines.
package lintutil // import "honnef.co/go/tools/lint/lintutil"

import (
	"errors"
	"flag"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"honnef.co/go/tools/lint"

	"github.com/kisielk/gotool"
	"golang.org/x/tools/go/loader"
)

func usage(name string, flags *flag.FlagSet) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", name)
		fmt.Fprintf(os.Stderr, "\t%s [flags] # runs on package in current directory\n", name)
		fmt.Fprintf(os.Stderr, "\t%s [flags] packages\n", name)
		fmt.Fprintf(os.Stderr, "\t%s [flags] directory\n", name)
		fmt.Fprintf(os.Stderr, "\t%s [flags] files... # must be a single package\n", name)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flags.PrintDefaults()
	}
}

type runner struct {
	checker lint.Checker
	tags    []string
	ignores []lint.Ignore

	unclean bool
}

func (runner runner) resolveRelative(importPaths []string) (goFiles bool, err error) {
	if len(importPaths) == 0 {
		return false, nil
	}
	if strings.HasSuffix(importPaths[0], ".go") {
		// User is specifying a package in terms of .go files, don't resolve
		return true, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	ctx := build.Default
	ctx.BuildTags = runner.tags
	for i, path := range importPaths {
		bpkg, err := ctx.Import(path, wd, build.FindOnly)
		if err != nil {
			return false, fmt.Errorf("can't load package %q: %v", path, err)
		}
		importPaths[i] = bpkg.ImportPath
	}
	return false, nil
}

func parseIgnore(s string) ([]lint.Ignore, error) {
	var out []lint.Ignore
	if len(s) == 0 {
		return nil, nil
	}
	for _, part := range strings.Fields(s) {
		p := strings.Split(part, ":")
		if len(p) != 2 {
			return nil, errors.New("malformed ignore string")
		}
		path := p[0]
		checks := strings.Split(p[1], ",")
		out = append(out, lint.Ignore{Pattern: path, Checks: checks})
	}
	return out, nil
}

func FlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = usage(name, flags)
	flags.Float64("min_confidence", 0, "Deprecated; use -ignore instead")
	flags.String("tags", "", "List of `build tags`")
	flags.String("ignore", "", "Space separated list of checks to ignore, in the following format: 'import/path/file.go:Check1,Check2,...' Both the import path and file name sections support globbing, e.g. 'os/exec/*_test.go'")
	flags.Bool("tests", true, "Include tests")
	return flags
}

func ProcessFlagSet(name string, c lint.Checker, fs *flag.FlagSet) {
	tags := fs.Lookup("tags").Value.(flag.Getter).Get().(string)
	ignore := fs.Lookup("ignore").Value.(flag.Getter).Get().(string)
	tests := fs.Lookup("tests").Value.(flag.Getter).Get().(bool)

	ignores, err := parseIgnore(ignore)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runner := &runner{
		checker: c,
		tags:    strings.Fields(tags),
		ignores: ignores,
	}
	paths := gotool.ImportPaths(fs.Args())
	goFiles, err := runner.resolveRelative(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		runner.unclean = true
	}
	ctx := build.Default
	ctx.BuildTags = runner.tags
	conf := &loader.Config{
		Build:      &ctx,
		ParserMode: parser.ParseComments,
		ImportPkgs: map[string]bool{},
	}
	if goFiles {
		conf.CreateFromFilenames("adhoc", paths...)
		lprog, err := conf.Load()
		if err != nil {
			log.Fatal(err)
		}
		ps := runner.lint(lprog)
		for _, ps := range ps {
			for _, p := range ps {
				runner.unclean = true
				fmt.Printf("%v: %s\n", relativePositionString(p.Position), p.Text)
			}
		}
	} else {
		for _, path := range paths {
			conf.ImportPkgs[path] = tests
		}
		lprog, err := conf.Load()
		if err != nil {
			log.Fatal(err)
		}
		ps := runner.lint(lprog)
		for _, ps := range ps {
			for _, p := range ps {
				runner.unclean = true
				fmt.Printf("%v: %s\n", relativePositionString(p.Position), p.Text)
			}

		}
	}
	if runner.unclean {
		os.Exit(1)
	}
}

func relativePositionString(pos token.Position) string {
	var s string
	pwd, err := os.Getwd()
	if err == nil {
		rel, err := filepath.Rel(pwd, pos.Filename)
		if err == nil {
			s = rel
		}
	}
	if s == "" {
		s = pos.Filename
	}
	if pos.IsValid() {
		if s != "" {
			s += ":"
		}
		s += fmt.Sprintf("%d:%d", pos.Line, pos.Column)
	}
	if s == "" {
		s = "-"
	}
	return s
}

func ProcessArgs(name string, c lint.Checker, args []string) {
	flags := FlagSet(name)
	flags.Parse(args)

	ProcessFlagSet(name, c, flags)
}

func (runner *runner) lint(lprog *loader.Program) map[string][]lint.Problem {
	l := &lint.Linter{
		Checker: runner.checker,
		Ignores: runner.ignores,
	}
	return l.Lint(lprog)
}
