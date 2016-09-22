// rdeps scans GOPATH for all reverse dependencies of a set of Go
// packages.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/build"
	"os"

	"github.com/kisielk/gotool"
	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/refactor/importgraph"
)

func main() {
	var tags buildutil.TagsFlag
	flag.Var(&tags, "tags", "List of build tags")
	stdin := flag.Bool("stdin", false, "Read packages from stdin instead of the command line")
	flag.Parse()

	ctx := build.Default
	ctx.BuildTags = tags
	var args []string
	if *stdin {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			args = append(args, s.Text())
		}
	} else {
		args = flag.Args()
	}
	if len(args) == 0 {
		return
	}
	pkgs := gotool.ImportPaths(args)
	_, reverse, errors := importgraph.Build(&ctx)
	_ = errors
	for _, pkg := range pkgs {
		for rdep := range reverse[pkg] {
			fmt.Println(rdep)
		}
	}
	for pkg, err := range errors {
		fmt.Fprintf(os.Stderr, "error in package %s: %s\n", pkg, err)
	}
}
