// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// irdump: a tool for displaying the IR form of Go programs.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"

	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/go/ir/irutil"

	"golang.org/x/tools/go/packages"
)

// flags
var (
	mode = ir.BuilderMode(0)

	testFlag = flag.Bool("test", false, "include implicit test packages and executables")

	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	tagsFlag = flag.String("tags", "", "comma-separated list of extra build tags (see: go help buildconstraint)")
)

func init() {
	flag.Var(&mode, "build", ir.BuilderModeDoc)
}

const usage = `SSA builder.
Usage: irdump [-build=[DBCSNFLG]] [-test] package...
Use -help flag to display options.

Examples:
% irdump -build=F hello.go              # dump SSA form of a single package
% irdump -build=F -test fmt             # dump SSA form of a package and its tests
`

func main() {
	if err := doMain(); err != nil {
		fmt.Fprintf(os.Stderr, "irdump: %s\n", err)
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
		BuildFlags: []string{"-tags=" + *tagsFlag},
		Mode:       packages.LoadSyntax,
		Tests:      *testFlag,
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

	// Load, parse and type-check the initial packages,
	// and, if -run, their dependencies.
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

	// Create (and display) SSA only for initial packages and wrappers.
	_, pkgs := irutil.Packages(initial, mode)
	for i, p := range pkgs {
		if p == nil {
			return fmt.Errorf("cannot build SSA for package %s", initial[i])
		}
		p.Build()
	}

	return nil
}
