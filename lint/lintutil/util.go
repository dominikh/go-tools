// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// Package lintutil provides helpers for writing linter command lines.
package lintutil // import "honnef.co/go/lint/lintutil"

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"strings"

	"honnef.co/go/lint"

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
	funcs         []lint.Func
	minConfidence float64
	tags          []string

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

func ProcessArgs(name string, funcs []lint.Func, args []string) {
	flags := &flag.FlagSet{}
	flags.Usage = usage(name, flags)
	var minConfidence = flags.Float64("min_confidence", 0.8, "minimum confidence of a problem to print it")
	var tags = flags.String("tags", "", "List of `build tags`")
	flags.Parse(args)

	runner := &runner{
		funcs:         funcs,
		minConfidence: *minConfidence,
		tags:          strings.Fields(*tags),
	}
	paths := gotool.ImportPaths(flags.Args())
	goFiles, err := runner.resolveRelative(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		runner.unclean = true
	}
	if goFiles {
		conf := &loader.Config{}
		conf.CreateFromFilenames("adhoc", paths...)
		lprog, err := conf.Load()
		if err != nil {
			log.Fatal(err)
		}
		ps := runner.lint(lprog)
		for _, ps := range ps {
			for _, p := range ps {
				runner.unclean = true
				if p.Confidence >= runner.minConfidence {
					fmt.Printf("%v: %s\n", p.Position, p.Text)
				}
			}
		}
	} else {
		ctx := build.Default
		conf := &loader.Config{
			Build: &ctx,
			TypeCheckFuncBodies: func(s string) bool {
				for _, path := range paths {
					if s == path || s == path+"_test" {
						return true
					}
				}
				return false
			},
		}
		for _, path := range paths {
			conf.ImportWithTests(path)
		}
		lprog, err := conf.Load()
		if err != nil {
			log.Fatal(err)
		}
		ps := runner.lint(lprog)
		for _, ps := range ps {
			for _, p := range ps {
				runner.unclean = true
				if p.Confidence >= runner.minConfidence {
					fmt.Printf("%v: %s\n", p.Position, p.Text)
				}
			}

		}
	}
	if runner.unclean {
		os.Exit(1)
	}
}

func (runner *runner) lint(lprog *loader.Program) map[string][]lint.Problem {
	l := &lint.Linter{
		Funcs: runner.funcs,
	}
	return l.Lint(lprog)
}
