---
title: Build tags
description: How to correctly check code that uses build tags
draft: true
---

## Introduction

In Go, files can have build tags, which control when said files will be part of a package.
For example, two files might contain alternate implementations of the same function, targeting Linux and Windows respectively.

Due to this, a single import path really refers to a collection of packages, with any particular package being chosen by a combination of build tags.
Even if your code doesn't make use of build tags, any of your transitive dependencies might.
Therefore, running `staticcheck my/package` really only checks one variant of `my/package`.

For more information on Go's build tags, see [`go help buildconstraint`](https://pkg.go.dev/cmd/go#hdr-Build_constraints).

## Implications

Checking packages using a single set of build tags can lead to both false positives and false negatives.
The reason for false negatives is straightforward: if some code is being excluded by build tags, then we won't check it.
False positives can be a bit more involved. Consider the following package, in [txtar format](https://pkg.go.dev/golang.org/x/tools/txtar#hdr-Txtar_format):

```go
-- api_linux.go --
package pkg

func Entry() { foo() }
-- api_windows.go --
package pkg

func Entry() { bar() }
-- shared.go --
package pkg

func foo() {}
func bar() {}
```

If we don't check the Windows build, then the function `bar` seems unused.
Similarly, if we don't check the Linux build, `foo` seems unused.

```terminal
$ GOOS=linux staticcheck
shared.go:4:6: func bar is unused (U1000)
$ GOOS=windows staticcheck
shared.go:3:6: func foo is unused (U1000)
```

Only when we check both builds do we see that both functions are in fact used.
Arguably, `foo` and `bar` should live in files with matching build tags to avoid this.
However, in reality, code bases can be complex, and it isn't always clear which sets of build tags make use of what code.
After all, if we always knew, we wouldn't have dead code to begin with.

Another example involves control flow. Consider this package, again in txtar format:

```go
-- api_linux.go --
package pkg

func dieIfUnsupported() { panic("unsupported") }
-- api_windows.go --
package pkg

func dieIfUnsupported() {}
-- shared.go --
package pkg

func foo() {}

func Entry() {
	dieIfUnsupported()
	foo()
}
```

Here, `dieIfUnsupported` panics unconditionally on Linux, but not on Windows.
Because Staticcheck takes control flow into consideration, this means that `foo` is unused on Linux but used on Windows.

Several checks have this sort of false positive, not just U1000.

## Solution

The solution to this problem is to run Staticcheck multiple times with different build tags and to merge the results.

At first glance, one might think that Staticcheck should be able to do this fully automatically: look at all build tags, find all unique combinations, and check them all.
However, this doesn't scale.
To be correct, Staticcheck would have to take dependencies and their tags into consideration, too.
Virtually all code depends on the Go standard library, and the Go standard library supports a plethora of operating systems, architectures, and a number of tags such as `netgo`.
All in all, there are thousands of unique combinations.
Checking all of these would take far too long.

However, the number of build configurations you care about is probably much smaller.
Your software probably support 2-3 operating systems on 1-2 architectures,
and maybe has a debug and a release build.
This makes for a lot fewer combinations that need to be checked.
These are probably the same combinations you're already checking in CI, too, by running their tests.
This will become useful in a bit.

### The `-merge` flag

Using the `-merge` flag, Staticcheck can merge the results of multiple runs.
It decides on a per-check basis whether any run or all runs have to have reported an issue for it to be valid.
It also takes into consideration which files were checked by which run, to reduce false negatives.

In order to use `-merge`, the runs to be merged have to use the `-f binary` flag.
This outputs results in a binary format containing all information required by `-merge`.
When using `-merge`, arguments are interpreted as file names instead of import paths, so that `staticcheck -merge file1 file2` will read the files `file1` and `file2`, which must contain the output of `staticcheck -f binary` runs, and merge them.

```terminal
$ GOOS=linux staticcheck -f binary >file1
$ GOOS=windows staticcheck -f binary >file2
$ staticcheck -merge file1 file2
...
```


Alternatively, if no arguments are passed, `staticcheck` will read from standard input instead.
This allows for workflows like

```
(
  GOOS=linux staticcheck -f binary
  GOOS=windows staticcheck -f binary
) | staticcheck -merge
```

This multi-step workflow of generating per-run output and merging it makes it possible to run Staticcheck on different systems before merging the results, which might be especially required when using cgo.

### The `-matrix` flag

With the `-matrix` flag, you can instruct Staticcheck to check multiple build configurations at once and merge the results.
In other words, it automates running Staticcheck multiple times and merging results afterwards.
This is useful when all configurations can be checked on a single system, for example because you don't use cgo.

When using the `-matrix` flag, Staticcheck reads a build matrix from standard input.
The build matrix uses a line-based format, where each line specifies a build name, environment variables and command-line flags.
Each line is of the format `<name>: [flags and environment...]`, for example `linux-debug: GOOS=linux -tags=debug`.
Environment variables and flags get passed to `go` when Staticcheck analyzes code, so you can use all flags that `go` supports, such as `-tags` or `-gcflags`, although few flags other than `-tags` are really useful.

<details>
<summary>Syntax rules of build matrices</summary>
Build names may consist of Unicode numbers, Unicode letters, and underscores.
Environment variable names may consist of a-z, A-Z, 0-9, and underscores.
Flag names may consist of a-z, 0-9, underscores, and dashes, and they must begin with a dash.
Values that contain spaces must be quoted using double quotes.
Flags with values must use equal signs. That is, <code>-tags=debug</code> is valid, but <code>-tags debug</code> is not.
Empty values are permitted.
Environment variables and flags can be freely mixed.
Empty lines are skipped.
</details>

Here is an example of using a build matrix:

```terminal
$ staticcheck -matrix <<EOF
windows: GOOS=windows
linux: GOOS=linux
appengine: GOOS=linux -tags=appengine
EOF
root_windows.go:292:47: syscall.StringToUTF16Ptr has been deprecated since Go 1.1: Use UTF16PtrFromString instead.  [windows] (SA1019)
verify_test.go:1338:7: const issuerSubjectMatchRoot is unused [appengine,linux,windows] (U1000)
```

Staticcheck will annotate results with the names of build configurations under which they occurred.

It's possible to combine `-matrix` and `-merge` by using `-matrix -f binary` and merging the results of multiple matrix runs.
