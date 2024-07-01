package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/tools/go/packages"
	"honnef.co/go/tools/go/loader"
	"honnef.co/go/tools/lintcmd/cache"
	"honnef.co/go/tools/unused"
)

// OPT(dh): we don't need full graph merging if we're not flagging exported objects. In that case, we can reuse the old
// list-based merging approach.

// OPT(dh): we can either merge graphs as we process packages, or we can merge them all in one go afterwards (then
// reloading them from cache). The first approach will likely lead to higher peak memory usage, but the latter may take
// more wall time to finish if we had spare CPU resources while processing packages.

func main() {
	opts := unused.DefaultOptions
	flag.BoolVar(&opts.FieldWritesAreUses, "field-writes-are-uses", opts.FieldWritesAreUses, "")
	flag.BoolVar(&opts.PostStatementsAreReads, "post-statements-are-reads", opts.PostStatementsAreReads, "")
	flag.BoolVar(&opts.ExportedIsUsed, "exported-is-used", opts.ExportedIsUsed, "")
	flag.BoolVar(&opts.ExportedFieldsAreUsed, "exported-fields-are-used", opts.ExportedFieldsAreUsed, "")
	flag.BoolVar(&opts.ParametersAreUsed, "parameters-are-used", opts.ParametersAreUsed, "")
	flag.BoolVar(&opts.LocalVariablesAreUsed, "local-variables-are-used", opts.LocalVariablesAreUsed, "")
	flag.BoolVar(&opts.GeneratedIsUsed, "generated-is-used", opts.GeneratedIsUsed, "")
	flag.Parse()

	// pprof.StartCPUProfile(os.Stdout)
	// defer pprof.StopCPUProfile()

	// XXX set cache key for this tool

	c, err := cache.Default()
	if err != nil {
		log.Fatal(err)
	}
	cfg := &packages.Config{
		Tests: true,
	}
	specs, err := loader.Graph(c, cfg, flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}

	var sg unused.SerializedGraph

	ourPkgs := map[string]struct{}{}

	for _, spec := range specs {
		if len(spec.Errors) != 0 {
			// XXX priunt errors
			continue
		}
		lpkg, _, err := loader.Load(spec, nil)
		if err != nil {
			continue
		}
		if len(lpkg.Errors) != 0 {
			continue
		}

		// XXX get directives and generated
		g := unused.Graph(lpkg.Fset, lpkg.Syntax, lpkg.Types, lpkg.TypesInfo, nil, nil, opts)
		sg.Merge(g)
		ourPkgs[spec.PkgPath] = struct{}{}
	}

	res := sg.Results()
	for _, obj := range res.Unused {
		// XXX format paths like staticcheck does
		if _, ok := ourPkgs[obj.Path.PkgPath]; !ok {
			continue
		}
		fmt.Printf("%s: %s %s is unused\n", obj.DisplayPosition, obj.Kind, obj.Name)
	}

	fmt.Fprintln(os.Stderr, sg.Dot())
}
