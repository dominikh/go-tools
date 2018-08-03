// staticcheck analyses Go code and makes it better.
package main // import "honnef.co/go/tools/cmd/staticcheck"

import (
	"fmt"
	"os"

	"honnef.co/go/tools/lint/lintutil"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
	"honnef.co/go/tools/unused"
)

const doc = `Usage of staticcheck:
	staticcheck [flags] [packages]

The staticcheck command runs code analysis on one or more packages and
reports bugs, stylistic issues, unused identifiers and more.

Packages are specified in the same way as for your underlying build
system (e.g. go build, or Bazel).


Checker categories

Staticcheck supports the following categories of checks:

- staticcheck: the original, name-giving category; checks for bugs,
  API misuses and critical performance issues.

- simple: checks for code that could be rewritten in a simpler manner.

- stylecheck: checks for the code's conformity with style and project
  guidelines.

- unused: checks for unused identifiers (types, functions, fields, …)


Selecting categories

By default, staticcheck runs all supported categories of checks. If
you want to select a subset of checks, you can do so with flags named
after the categories:

	-simple
	-staticcheck
	-stylecheck
	-unused

When using these flags, only the categories explicitly specified will be
run.


Exiting with non-zero status code

Staticcheck can exit with a non-zero code if it finds any issues. By
default, it only exits non-zero for a predefined subset of checks. This is
because people run staticcheck as part of their CI/code review
pipeline, and not all issues should fail a build. By default, issues
found in the 'staticcheck' and 'unused' categories cause non-zero
exiting.

If you wish to control which categories cause non-zero exiting, use the
'-<category>.exit-non-zero' bool flags.
`

func usage() {
	fmt.Fprint(os.Stderr, doc)
}

func main() {
	var flags struct {
		staticcheck struct {
			enabled     bool
			exitNonZero bool
		}
		simple struct {
			enabled     bool
			exitNonZero bool
		}
		unused struct {
			enabled     bool
			exitNonZero bool
		}
		stylecheck struct {
			enabled     bool
			exitNonZero bool
		}
	}

	fs := lintutil.FlagSet("staticcheck")
	fs.Usage = usage
	fs.BoolVar(&flags.simple.enabled, "simple", false, "Enable 'simple' category of checks")
	fs.BoolVar(&flags.staticcheck.enabled, "staticcheck", false, "Enable 'staticcheck' category of checks")
	fs.BoolVar(&flags.stylecheck.enabled, "stylecheck", false, "Enable 'Stylecheck' category of checks")
	fs.BoolVar(&flags.unused.enabled, "unused", false, "Enable 'unused' category of checks")

	fs.BoolVar(&flags.simple.exitNonZero, "simple.exit-non-zero", false, "Exit non-zero if any problems were found")
	fs.BoolVar(&flags.staticcheck.exitNonZero, "staticcheck.exit-non-zero", true, "Exit non-zero if any problems were found")
	fs.BoolVar(&flags.stylecheck.exitNonZero, "stylecheck.exit-non-zero", false, "Exit non-zero if any problems were found")
	fs.BoolVar(&flags.unused.exitNonZero, "unused.exit-non-zero", true, "Exit non-zero if any problems were found")

	fs.Parse(os.Args[1:])

	if !flags.simple.enabled && !flags.staticcheck.enabled && !flags.stylecheck.enabled && !flags.unused.enabled {
		flags.simple.enabled = true
		flags.staticcheck.enabled = true
		flags.stylecheck.enabled = true
		flags.unused.enabled = true
	}

	var checkers []lintutil.CheckerConfig

	if flags.simple.enabled {
		sc := simple.NewChecker()
		sc.CheckGenerated = false
		checkers = append(checkers, lintutil.CheckerConfig{
			Checker:     sc,
			ExitNonZero: flags.simple.exitNonZero,
		})
	}

	if flags.staticcheck.enabled {
		sac := staticcheck.NewChecker()
		sac.CheckGenerated = true
		checkers = append(checkers, lintutil.CheckerConfig{
			Checker:     sac,
			ExitNonZero: flags.staticcheck.exitNonZero,
		})
	}

	if flags.stylecheck.enabled {
		stc := stylecheck.NewChecker()
		stc.CheckGenerated = true
		checkers = append(checkers, lintutil.CheckerConfig{
			Checker:     stc,
			ExitNonZero: flags.stylecheck.exitNonZero,
		})
	}

	if flags.unused.enabled {
		var mode unused.CheckMode
		uc := unused.NewChecker(mode)
		uc.ConsiderReflection = true
		checkers = append(checkers, lintutil.CheckerConfig{
			Checker:     unused.NewLintChecker(uc),
			ExitNonZero: flags.unused.exitNonZero,
		})

	}

	lintutil.ProcessFlagSet(checkers, fs)
}
