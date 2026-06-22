// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ir_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"

	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/go/ir/irutil"

	"golang.org/x/tools/go/packages"
)

const hello = `
package main

import "fmt"

const message = "Hello, World!"

func main() {
	fmt.Println(message)
}
`

// This program demonstrates how to run the IR builder on a single
// package of one or more already-parsed files.  Its dependencies are
// loaded from compiler export data.  This is what you'd typically use
// for a compiler; it does not depend on golang.org/x/tools/go/loader.
//
// It shows the printed representation of packages, functions, and
// instructions.  Within the function listing, the name of each
// BasicBlock such as ".0.entry" is printed left-aligned, followed by
// the block's Instructions.
//
// For each instruction that defines an IR virtual register
// (i.e. implements Value), the type of that value is shown in the
// right column.
//
// Build and run the irdump.go program if you want a standalone tool
// with similar functionality. It is located at
// honnef.co/go/tools/internal/cmd/irdump.
func Example_buildPackage() {
	// Parse the source files.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hello.go", hello, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		fmt.Print(err) // parse error
		return
	}
	files := []*ast.File{f}

	// Create the type-checker's package.
	pkg := types.NewPackage("hello", "")

	// Type-check the package, load dependencies.
	// Create and build the IR program.
	hello, _, err := irutil.BuildPackage(
		&types.Config{Importer: importer.Default()}, fset, pkg, files, ir.SanityCheckFunctions)
	if err != nil {
		fmt.Print(err) // type error in some package
		return
	}

	// Print out the package.
	hello.WriteTo(os.Stdout)

	// Print out the package-level functions.
	// Replace interface{} with any so the tests work for Go 1.17 and Go 1.18.
	{
		var buf bytes.Buffer
		ir.WriteFunction(&buf, hello.Func("init"))
		fmt.Print(strings.ReplaceAll(buf.String(), "interface{}", "any"))
	}
	{
		var buf bytes.Buffer
		ir.WriteFunction(&buf, hello.Func("main"))
		fmt.Print(strings.ReplaceAll(buf.String(), "interface{}", "any"))
	}

	// Output:
	// package hello:
	//   func  init       func()
	//   var   init$guard bool
	//   func  main       func()
	//   const message    message = "Hello, World!":untyped string
	//
	// # Name: hello.init
	// # Package: hello
	// # Synthetic: package initializer
	// func init():
	// 0:                                                                entry P:0 S:2
	//         t1 = *init$guard                                                   bool
	//         if t1 goto 2 else 1
	// 1:                                                    init.start P:1 S:1 idom:0
	//         *init$guard = true:bool
	//         t4 = fmt.init()                                                      ()
	//         jump 2
	// 2:                                                     init.done P:2 S:0 idom:0
	//         return
	//
	// # Name: hello.main
	// # Package: hello
	// # Location: hello.go:8:1
	// func main():
	// 0:                                                                entry P:0 S:0
	//         t1 = new [1]any (varargs)                                       *[1]any
	//         t2 = &t1[0:int]                                                    *any
	//         t3 = make any <- string ("Hello, World!":string)                    any
	//         *t2 = t3
	//         t5 = slice t1[:]                                                  []any
	//         t6 = fmt.Println(t5...)                              (n int, err error)
	//         return
}

// This example builds IR code for a set of packages using the
// x/tools/go/packages API. This is what you would typically use for a
// analysis capable of operating on a single package.
func Example_loadPackages() {
	// Load, parse, and type-check the initial packages.
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo}
	initial, err := packages.Load(cfg, "fmt", "net/http")
	if err != nil {
		log.Fatal(err)
	}

	// Stop if any package had errors.
	// This step is optional; without it, the next step
	// will create IR for only a subset of packages.
	if packages.PrintErrors(initial) > 0 {
		log.Fatalf("packages contain errors")
	}

	// Create IR packages for all well-typed packages.
	prog, pkgs := irutil.Packages(initial, ir.PrintPackages)
	_ = prog

	// Build IR code for the well-typed initial packages.
	for _, p := range pkgs {
		if p != nil {
			p.Build()
		}
	}
}

// This example builds IR code for a set of packages plus all their dependencies,
// using the x/tools/go/packages API.
// This is what you'd typically use for a whole-program analysis.
func Example_loadWholeProgram() {
	// Load, parse, and type-check the whole program.
	cfg := packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedDeps}
	initial, err := packages.Load(&cfg, "fmt", "net/http")
	if err != nil {
		log.Fatal(err)
	}

	// Create IR packages for well-typed packages and their dependencies.
	prog, pkgs := irutil.AllPackages(initial, ir.PrintPackages)
	_ = pkgs

	// Build IR code for the whole program.
	prog.Build()
}
