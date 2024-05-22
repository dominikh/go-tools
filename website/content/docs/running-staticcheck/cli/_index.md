---
title: Command-line interface
description: How to use the `staticcheck` command
---
The `staticcheck` command is the primary way of running Staticcheck.

At its core, the `staticcheck` command works a lot like `go vet` or `go build`.
It accepts the same package patterns (see `go help packages` for details),
it outputs problems in the same format,
it supports a `-tags` flag for specifying which build tags to use, and so on.
Overall, it is meant to feel like another `go` command.

However, it also comes with several of its own flags to support some of its unique functionality.
This article will focus on explaining that unique functionality.

<!-- TODO -->
<!-- ## Specifying which checks to run {#checks} -->

## Explaining checks {#explain}

You can use `staticcheck -explain <check>` to get a helpful description of a check.

Every diagnostic that staticcheck reports is annotated with the identifier of the specific check that found the issue. For example, in

```text
foo.go:1248:4: unnecessary use of fmt.Sprintf (S1039)
```

the check's identifier is S1039. Running `staticcheck -explain S1039` will output the following:

```text
Unnecessary use of fmt.Sprint

Calling fmt.Sprint with a single string argument is unnecessary and identical to using the string directly.

Available since
	2020.1

Online documentation
	https://staticcheck.dev/docs/checks#S1039
```

The output includes a one-line summary, one or more paragraphs of helpful text, the first version of Staticcheck that the check appeared in, and a link to online documentation, which contains the same information as the output of `staticcheck -explain`.

## Selecting an output format {#format}

Staticcheck can format its output in a number of ways, by using the `-f` flag.
See this [list of formatters]({{< relref "/docs/running-staticcheck/cli/formatters" >}}) for a list of all formatters.

<!-- TODO -->
<!-- ## Controlling the exit status {#fail} -->

## Excluding tests {#tests}

By default, Staticcheck analyses packages as well as their tests.
By passing `-tests=false`, one can skip the analysis of tests.
This is primarily useful for the {{< check "U1000" >}} check, as it allows finding code that is only used by tests and would otherwise be unused.
