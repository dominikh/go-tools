// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package buildssa defines an Analyzer that constructs the SSA
// representation of an error-free package and returns the set of all
// functions within it. It does not report any diagnostics itself but
// may be used as an input to other analyzers.
//
// THIS INTERFACE IS EXPERIMENTAL AND MAY BE SUBJECT TO INCOMPATIBLE CHANGE.
package buildssa

import (
	"go/ast"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/ssa"
)

type willExit struct{}
type willUnwind struct{}

func (*willExit) AFact()   {}
func (*willUnwind) AFact() {}

var Analyzer = &analysis.Analyzer{
	Name:       "buildssa",
	Doc:        "build SSA-form IR for later passes",
	Run:        run,
	ResultType: reflect.TypeOf(new(SSA)),
	FactTypes:  []analysis.Fact{new(willExit), new(willUnwind)},
}

// SSA provides SSA-form intermediate representation for all the
// non-blank source functions in the current package.
type SSA struct {
	Pkg      *ssa.Package
	SrcFuncs []*ssa.Function
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Plundered from ssautil.BuildPackage.

	// We must create a new Program for each Package because the
	// analysis API provides no place to hang a Program shared by
	// all Packages. Consequently, SSA Packages and Functions do not
	// have a canonical representation across an analysis session of
	// multiple packages. This is unlikely to be a problem in
	// practice because the analysis API essentially forces all
	// packages to be analysed independently, so any given call to
	// Analysis.Run on a package will see only SSA objects belonging
	// to a single Program.

	mode := ssa.GlobalDebug

	prog := ssa.NewProgram(pass.Fset, mode)

	// Create SSA packages for all imports.
	// Order is not significant.
	created := make(map[*types.Package]bool)
	var createAll func(pkgs []*types.Package)
	createAll = func(pkgs []*types.Package) {
		for _, p := range pkgs {
			if !created[p] {
				created[p] = true
				ssapkg := prog.CreatePackage(p, nil, nil, true)
				for _, fn := range ssapkg.Functions {
					if ast.IsExported(fn.Name()) {
						var exit willExit
						var unwind willUnwind
						if pass.ImportObjectFact(fn.Object(), &exit) {
							fn.WillExit = true
						}
						if pass.ImportObjectFact(fn.Object(), &unwind) {
							fn.WillUnwind = true
						}
					}
				}
				createAll(p.Imports())
			}
		}
	}
	createAll(pass.Pkg.Imports())

	// Create and build the primary package.
	ssapkg := prog.CreatePackage(pass.Pkg, pass.Files, pass.TypesInfo, false)
	ssapkg.Build()

	// Compute list of source functions, including literals,
	// in source order.
	var addAnons func(f *ssa.Function)
	funcs := make([]*ssa.Function, len(ssapkg.Functions))
	copy(funcs, ssapkg.Functions)
	addAnons = func(f *ssa.Function) {
		for _, anon := range f.AnonFuncs {
			funcs = append(funcs, anon)
			addAnons(anon)
		}
	}
	for _, fn := range ssapkg.Functions {
		addAnons(fn)
		if fn.WillExit {
			pass.ExportObjectFact(fn.Object(), new(willExit))
		}
		if fn.WillUnwind {
			pass.ExportObjectFact(fn.Object(), new(willUnwind))
		}
	}

	return &SSA{Pkg: ssapkg, SrcFuncs: funcs}, nil
}
