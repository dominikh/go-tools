// megacheck runs staticcheck, gosimple and unused.
package main // import "honnef.co/go/tools/cmd/megacheck"

import (
	"fmt"
	"os"

	"honnef.co/go/tools/lint"
	"honnef.co/go/tools/lint/lintutil"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/unused"
)

func main() {
	fmt.Fprintln(os.Stderr, "Megacheck has been deprecated. Please use staticcheck instead.")

	var flags struct {
		staticcheck struct {
			enabled   bool
			generated bool
		}
		gosimple struct {
			enabled   bool
			generated bool
		}
		unused struct {
			enabled      bool
			constants    bool
			fields       bool
			functions    bool
			types        bool
			variables    bool
			wholeProgram bool
			reflection   bool
		}
	}
	fs := lintutil.FlagSet("megacheck")
	fs.BoolVar(&flags.gosimple.enabled,
		"simple.enabled", true, "Deprecated: use -checks instead")
	fs.BoolVar(&flags.gosimple.generated,
		"simple.generated", false, "Check generated code")

	fs.BoolVar(&flags.staticcheck.enabled,
		"staticcheck.enabled", true, "Deprecated: use -checks instead")
	fs.BoolVar(&flags.staticcheck.generated,
		"staticcheck.generated", false, "Check generated code (only applies to a subset of checks)")

	fs.BoolVar(&flags.unused.enabled,
		"unused.enabled", true, "Deprecated: use -checks instead")
	fs.BoolVar(&flags.unused.constants,
		"unused.consts", true, "Report unused constants")
	fs.BoolVar(&flags.unused.fields,
		"unused.fields", true, "Report unused fields")
	fs.BoolVar(&flags.unused.functions,
		"unused.funcs", true, "Report unused functions and methods")
	fs.BoolVar(&flags.unused.types,
		"unused.types", true, "Report unused types")
	fs.BoolVar(&flags.unused.variables,
		"unused.vars", true, "Report unused variables")
	fs.BoolVar(&flags.unused.wholeProgram,
		"unused.exported", false, "Treat arguments as a program and report unused exported identifiers")
	fs.BoolVar(&flags.unused.reflection,
		"unused.reflect", true, "Consider identifiers as used when it's likely they'll be accessed via reflection")

	fs.Bool("simple.exit-non-zero", true, "Deprecated: use -fail instead")
	fs.Bool("staticcheck.exit-non-zero", true, "Deprecated: use -fail instead")
	fs.Bool("unused.exit-non-zero", true, "Deprecated: use -fail instead")

	fs.Parse(os.Args[1:])

	var checkers []lint.Checker

	if flags.staticcheck.enabled {
		sac := staticcheck.NewChecker()
		sac.CheckGenerated = flags.staticcheck.generated
		checkers = append(checkers, sac)
	}

	if flags.gosimple.enabled {
		sc := simple.NewChecker()
		sc.CheckGenerated = flags.gosimple.generated
		checkers = append(checkers, sc)
	}

	if flags.unused.enabled {
		uc := &unused.Checker{}
		uc.WholeProgram = flags.unused.wholeProgram
		checkers = append(checkers, uc)
	}

	lintutil.ProcessFlagSet(checkers, fs)
}
