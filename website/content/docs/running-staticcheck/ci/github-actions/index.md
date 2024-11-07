---
title: GitHub Actions
description: Running Staticcheck in GitHub Actions
aliases:
  - /docs/running-staticcheck/github-actions
---
We publish [our own action](https://github.com/marketplace/actions/staticcheck) for [GitHub Actions](https://github.com/features/actions),
which makes it very simple to run Staticcheck in CI on GitHub.

## Examples

At its simplest, just add `dominikh/staticcheck-action` as a step in your existing workflow.
A minimal workflow might look like this:

```yaml
name: "CI"
on: ["push", "pull_request"]

jobs:
  ci:
    name: "Run CI"
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
      with:
        fetch-depth: 1
    - uses: dominikh/staticcheck-action@v1.3.0
      with:
        version: "2023.1.1"
```

A more advanced example that runs tests, go vet and Staticcheck on multiple OSs and Go versions looks like this:

```yaml
name: "CI"
on: ["push", "pull_request"]

jobs:
  ci:
    name: "Run CI"
    strategy:
      fail-fast: false
      matrix:
        os: ["windows-latest", "ubuntu-latest", "macOS-latest"]
        go: ["1.16.x", "1.17.x"]
    runs-on: ${{ matrix.os }}
    steps:
    - uses: actions/checkout@v3
      with:
        fetch-depth: 1
    - uses: WillAbides/setup-go-faster@v1.8.0
      with:
        go-version: ${{ matrix.go }}
    - run: "go test ./..."
    - run: "go vet ./..."
    - uses: dominikh/staticcheck-action@v1.3.0
      with:
        version: "2023.1.1"
        install-go: false
        cache-key: ${{ matrix.go }}
```

Note that this example could benefit from further improvements, such as caching of Go's build cache.

## Managing Go

By default, `staticcheck-action` installs Go so that it can install and run Staticcheck.
It also saves and restores Go's build cache (in addition to Staticcheck's own cache) to speed up future runs.
This is intended for trivial jobs that only run Staticcheck, not other steps such as `go test`.

For more complicated jobs, it is strongly recommended that you set `install-go` to `false`,
install Go yourself (e.g. by using [`actions/setup-go`](https://github.com/actions/setup-go) or [`WillAbides/setup-go-faster`](https://github.com/WillAbides/setup-go-faster)),
and save and restore the Go build cache, for improved performance.

When installing Go, make sure the version meets Staticcheck's minimum requirements.
A given Staticcheck release supports the last two versions of Go (such as Go 1.16 and Go 1.17) at the time of release.
The action itself requires at least Go 1.16.

## Options

### `version`

Which version of Staticcheck to use.
Because new versions of Staticcheck introduce new checks that may break your build,
it is recommended to pin to a specific version and to update Staticheck consciously.

It defaults to `latest`, which installs the latest released version of Staticcheck.

### `min-go-version`

Minimum version of Go to support. This affects the diagnostics reported by Staticcheck,
avoiding suggestions that are not applicable to older versions of Go.

If unset, this will default to the Go version specified in your go.mod.

See https://staticcheck.dev/docs/running-staticcheck/cli/#go for more information.

### `build-tags`

Go build tags that get passed to Staticcheck via the `-tags` flag.

### `install-go`

Whether the action should install a suitable version of Go to install and run Staticcheck.
If Staticcheck is the only action in your job, this option can usually be left on its default value of `true`.
If your job already installs Go prior to running Staticcheck, for example to run unit tests, it is best to set this option to `false`.

The latest release of Staticcheck works with the last two minor releases of Go.
The action itself requires at least Go 1.16.

### `cache-key`

String to include in the cache key, in addition to the default, which is `runner.os`.
This is useful when using multiple Go versions.
