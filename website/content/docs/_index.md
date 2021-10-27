---
title: "Welcome to Staticcheck"
linkTitle: "Documentation"
menu:
  main:
    weight: 1
    pre: <i class='fas fa-book'></i>
---

Staticcheck is a state of the art linter for the [Go programming language](https://go.dev/).
Using static analysis, it finds bugs and performance issues, offers simplifications, and enforces style rules.


Each of the
[
{{< numchecks.inline >}}
{{- $c := len $.Site.Data.checks.Checks -}}
{{- sub $c (mod $c 25) -}}
{{< /numchecks.inline >}}+
]({{< relref "/docs/checks" >}}) checks has been designed to be fast, precise and useful.
When Staticcheck flags code, you can be sure that it isn't wasting your time with unactionable warnings.
Unlike many other linters, Staticcheck focuses on checks that produce few to no false positives.
It's the ideal candidate for running in CI without risking spurious failures.

Staticcheck aims to be trivial to adopt.
It behaves just like the official `go` tool and requires no learning to get started with.
Just run `staticcheck ./...` on your code in addition to `go vet ./...`.

While checks have been designed to be useful out of the box,
they still provide [configuration]({{< relref "/docs/configuration" >}}) where necessary, to fine-tune to your needs, without overwhelming you with hundreds of options.

Staticcheck can be used from the command line, in CI,
and even [directly from your editor](https://github.com/golang/tools/blob/master/gopls/doc/settings.md#staticcheck-bool).


Staticcheck is open source and offered completely free of charge. [Sponsors]({{< relref "/sponsors" >}}) guarantee its continued development.

<link rel="prefetch" href="{{< relref "/docs/getting-started" >}}">
