// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssa_test

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"

	"golang.org/x/tools/go/packages"
	"honnef.co/go/tools/ssa"
	"honnef.co/go/tools/ssa/ssautil"
)

const hello = `
package main

import "fmt"

const message = "Hello, World!"

func main() {
	fmt.Println(message)
}
`

// This program demonstrates how to run the SSA builder on a single
// package of one or more already-parsed files.  Its dependencies are
// loaded from compiler export data.  This is what you'd typically use
// for a compiler; it does not depend on golang.org/x/tools/go/loader.
//
// It shows the printed representation of packages, functions, and
// instructions.  Within the function listing, the name of each
// BasicBlock such as ".0.entry" is printed left-aligned, followed by
// the block's Instructions.
//
// For each instruction that defines an SSA virtual register
// (i.e. implements Value), the type of that value is shown in the
// right column.
//
// Build and run the ssadump.go program if you want a standalone tool
// with similar functionality. It is located at
// golang.org/x/tools/cmd/ssadump.
//
func Example_buildPackage() {
	// Parse the source files.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hello.go", hello, parser.ParseComments)
	if err != nil {
		fmt.Print(err) // parse error
		return
	}
	files := []*ast.File{f}

	// Create the type-checker's package.
	pkg := types.NewPackage("hello", "")

	// Type-check the package, load dependencies.
	// Create and build the SSA program.
	hello, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()}, fset, pkg, files, ssa.SanityCheckFunctions)
	if err != nil {
		fmt.Print(err) // type error in some package
		return
	}

	// Print out the package.
	hello.WriteTo(os.Stdout)

	// Print out the package-level functions.
	hello.Func("init").WriteTo(os.Stdout)
	hello.Func("main").WriteTo(os.Stdout)

	// Output:
	// package hello:
	//   func  init       func()
	//   var   init$guard bool
	//   func  main       func()
	//   const message    message = Const <untyped string> {"Hello, World!"}
	//
	// # Name: hello.init
	// # Package: hello
	// # Synthetic: package initializer
	// func init():
	// b0:
	// 	t1 = Const <bool> {true}
	// 	t2 = Load <bool> init$guard
	// 	If t2 → b1 b2
	//
	// b1: ← b0 b2
	// 	Return
	//
	// b2: ← b0
	// 	Store {bool} init$guard t1
	// 	t6 = Call <()> fmt.init
	// 	Jump → b1
	//
	// # Name: hello.main
	// # Package: hello
	// # Location: hello.go:8:6
	// func main():
	// b0:
	// 	t1 = Const <string> {"Hello, World!"}
	// 	t2 = Const <int> {0}
	// 	t3 = HeapAlloc <*[1]interface{}> (varargs)
	// 	t4 = IndexAddr <*interface{}> t3 t2
	// 	t5 = MakeInterface <interface{}> t1
	// 	Store {interface{}} t4 t5
	// 	t7 = Slice <[]interface{}> t3 <nil> <nil> <nil>
	// 	t8 = Call <(n int, err error)> fmt.Println t7
	// 	Jump → b1
	//
	// b1: ← b0
	// 	Return
}

// This example builds SSA code for a set of packages using the
// x/tools/go/packages API. This is what you would typically use for a
// analysis capable of operating on a single package.
func Example_loadPackages() {
	// Load, parse, and type-check the initial packages.
	cfg := &packages.Config{Mode: packages.LoadSyntax}
	initial, err := packages.Load(cfg, "fmt", "net/http")
	if err != nil {
		log.Fatal(err)
	}

	// Stop if any package had errors.
	// This step is optional; without it, the next step
	// will create SSA for only a subset of packages.
	if packages.PrintErrors(initial) > 0 {
		log.Fatalf("packages contain errors")
	}

	// Create SSA packages for all well-typed packages.
	prog, pkgs := ssautil.Packages(initial, ssa.PrintPackages, nil)
	_ = prog

	// Build SSA code for the well-typed initial packages.
	for _, p := range pkgs {
		if p != nil {
			p.Build()
		}
	}
}

// This example builds SSA code for a set of packages plus all their dependencies,
// using the x/tools/go/packages API.
// This is what you'd typically use for a whole-program analysis.
func Example_loadWholeProgram() {
	// Load, parse, and type-check the whole program.
	cfg := packages.Config{Mode: packages.LoadAllSyntax}
	initial, err := packages.Load(&cfg, "fmt", "net/http")
	if err != nil {
		log.Fatal(err)
	}

	// Create SSA packages for well-typed packages and their dependencies.
	prog, pkgs := ssautil.AllPackages(initial, ssa.PrintPackages, nil)
	_ = pkgs

	// Build SSA code for the whole program.
	prog.Build()
}
