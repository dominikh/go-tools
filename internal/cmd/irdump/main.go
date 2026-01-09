// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// irdump: a tool for displaying the IR form of Go programs.
package main

import (
	"flag"

	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/internal/passes/buildir"

	"golang.org/x/tools/go/analysis/singlechecker"
)

// flags
var (
	dot  bool
	html string
)

func init() {
	flag.BoolVar(&dot, "dot", false, "Print Graphviz dot of CFG")
	flag.StringVar(&html, "html", "", "Print HTML for 'function'")
}

func main() {
	buildir.Debug.Mode = ir.PrintFunctions | ir.PrintPackages
	flag.Func("build", ir.BuilderModeDoc, func(s string) error {
		return buildir.Debug.Mode.Set(s)
	})
	singlechecker.Main(buildir.Analyzer)

	// if dot {
	// 	for _, p := range pkgs {
	// 		for _, m := range p.Members {
	// 			if fn, ok := m.(*ir.Function); ok {
	// 				fmt.Println("digraph{")
	// 				fmt.Printf("label = %q;\n", fn.Name())
	// 				for _, b := range fn.Blocks {
	// 					fmt.Printf("n%d [label=\"%d: %s\"]\n", b.Index, b.Index, b.Comment)
	// 					for _, succ := range b.Succs {
	// 						fmt.Printf("n%d -> n%d\n", b.Index, succ.Index)
	// 					}
	// 				}
	// 				fmt.Println("}")
	// 			}
	// 		}
	// 	}
	// }
}
