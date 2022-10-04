---
title: "Getting started"
description: "Quickly get started using Staticcheck"
weight: 1
aliases:
  - /docs/install
---

## Installation

Beginning with Go 1.17, the simplest way of installing Staticcheck is by running `go install honnef.co/go/tools/cmd/staticcheck@latest`.
This will install the latest version of Staticcheck to `$GOPATH/bin`. To find out where `$GOPATH` is, run `go env GOPATH`.
Instead of `@latest`, you can also use a specific version, such as `@2020.2.1`.

If you'd like to be notified of new releases, you can use [GitHub's Releases only watches](https://docs.github.com/en/github/managing-subscriptions-and-notifications-on-github/viewing-your-subscriptions#configuring-your-watch-settings-for-an-individual-repository).

### Binary releases

We publish binary releases for the most common operating systems and CPU architectures.
These can be downloaded from [GitHub](https://github.com/dominikh/go-tools/releases).

### Distribution packages

Many package managers include Staticcheck, allowing you to install it with your usual commands, such as `apt install`.
Note, however, that you might not always get new releases in a timely manner.

What follows is a non-exhaustive list of the package names in various package repositories.

<div id="getting-started-distribution-packages">

Arch Linux
: [staticcheck](https://archlinux.org/packages/community/x86_64/staticcheck/)

Debian
: [go-staticcheck](https://packages.debian.org/go-staticcheck)

Fedora
: [golang-honnef-tools](https://fedora.pkgs.org/33/fedora-x86_64/golang-honnef-tools-2020.1.5-2.fc33.x86_64.rpm.html)

Homebrew
: [staticcheck](https://formulae.brew.sh/formula/staticcheck)

MacPorts
: [staticcheck](https://ports.macports.org/port/staticcheck/summary)

NixOS
: go-tools

Scoop
: [staticcheck](https://github.com/ScoopInstaller/Main/blob/master/bucket/staticcheck.json)

</div>

## Running Staticcheck

The `staticcheck` command works much like `go build` or `go vet` do.
It supports all of the same package patterns.
For example, `staticcheck .` will check the current package, and `staticcheck ./...` will check all packages.
For more details on specifying packages to check, see `go help packages`.

Therefore, to start using Staticcheck, just run it on your code: `staticcheck ./...`.
It will print any issues it finds, or nothing at all if your code is squeaky clean.

Read the [Running Staticcheck]({{< relref "/docs/running-staticcheck" >}}) articles to learn more about running Staticcheck.
