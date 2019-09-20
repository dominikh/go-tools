// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// ssadump: a tool for displaying the SSA form of Go programs.
package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"runtime/pprof"

	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/packages"
	"honnef.co/go/tools/ssa"
	"honnef.co/go/tools/ssa/ssautil"
)

// flags
var (
	mode       = ssa.BuilderMode(0)
	testFlag   = flag.Bool("test", false, "include implicit test packages and executables")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	dot        bool
	html       string
)

func init() {
	flag.Var(&mode, "build", ssa.BuilderModeDoc)
	flag.Var((*buildutil.TagsFlag)(&build.Default.BuildTags), "tags", buildutil.TagsFlagDoc)
	flag.BoolVar(&dot, "dot", false, "Print Graphviz dot of CFG")
	flag.StringVar(&html, "html", "", "Print HTML for 'function'")
}

const usage = `SSA builder.
Usage: ssadump [-build=[DBCSNFL]] [-test] [-arg=...] package...
Use -help flag to display options.

Examples:
% ssadump -build=F hello.go              # dump SSA form of a single package
% ssadump -build=F -test fmt             # dump SSA form of a package and its tests
`

func main() {
	if err := doMain(); err != nil {
		fmt.Fprintf(os.Stderr, "ssadump: %s\n", err)
		os.Exit(1)
	}
}

func doMain() error {
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: *testFlag,
	}

	// Profiling support.
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Load, parse and type-check the initial packages.
	initial, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		return err
	}
	if len(initial) == 0 {
		return fmt.Errorf("no packages")
	}
	if packages.PrintErrors(initial) > 0 {
		return fmt.Errorf("packages contain errors")
	}

	// Create SSA-form program representation.
	_, pkgs := ssautil.Packages(initial, mode, &ssautil.Options{PrintFunc: html})

	for i, p := range pkgs {
		if p == nil {
			return fmt.Errorf("cannot build SSA for package %s", initial[i])
		}
	}

	// Build and display only the initial packages
	// (and synthetic wrappers).
	for _, p := range pkgs {
		p.Build()
	}

	if dot {
		for _, p := range pkgs {
			for _, m := range p.Members {
				if fn, ok := m.(*ssa.Function); ok {
					fmt.Println("digraph{")
					fmt.Printf("label = %q;\n", fn.Name())
					for _, b := range fn.Blocks {
						fmt.Printf("n%d [label=\"%d: %s\"]\n", b.Index, b.Index, b.Comment)
						for _, succ := range b.Succs {
							fmt.Printf("n%d -> n%d\n", b.Index, succ.Index)
						}
					}
					fmt.Println("}")
				}
			}
		}
	}
	return nil
}
