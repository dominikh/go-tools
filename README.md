<div align="center">
	<h1><img alt="Staticcheck logo" src="/images/logo.svg" height="300" /><br />
		The advanced Go linter
	</h1>
</div>

Staticcheck is a state of the art linter for the [Go programming
language](https://go.dev/). Using static analysis, it finds bugs and performance issues,
offers simplifications, and enforces style rules.

**Financial support by [private and corporate sponsors](http://staticcheck.io/sponsors) guarantees the tool's continued development.
Please [become a sponsor](https://github.com/users/dominikh/sponsorship) if you or your company rely on Staticcheck.**


## Documentation

You can find extensive documentation on Staticcheck on [its website](https://staticcheck.io/docs/).

## Installation

### Releases

It is recommended that you run released versions of the tools.
These releases can be found as git tags (e.g. `2022.1`).

The easiest way of installing a release is by using `go install`, for example `go install honnef.co/go/tools/cmd/staticcheck@2022.1`.
Alternatively, we also offer [prebuilt binaries](https://github.com/dominikh/go-tools/releases).

You can find more information about installation and releases in the [documentation](https://staticcheck.io/docs/getting-started/).

### Master

You can also run the master branch instead of a release. Note that
while the master branch is usually stable, it may still contain new
checks or backwards incompatible changes that break your build. By
using the master branch you agree to become a beta tester.

## Tools

All of the following tools can be found in the cmd/ directory. Each
tool is accompanied by its own README, describing it in more detail.

| Tool                                               | Description                                                             |
|----------------------------------------------------|-------------------------------------------------------------------------|
| [keyify](cmd/keyify/)                              | Transforms an unkeyed struct literal into a keyed one.                  |
| [staticcheck](cmd/staticcheck/)                    | Go static analysis, detecting bugs, performance issues, and much more. |
| [structlayout](cmd/structlayout/)                  | Displays the layout (field sizes and padding) of structs.               |
| [structlayout-optimize](cmd/structlayout-optimize) | Reorders struct fields to minimize the amount of padding.               |
| [structlayout-pretty](cmd/structlayout-pretty)     | Formats the output of structlayout with ASCII art.                      |

## Libraries

In addition to the aforementioned tools, this repository contains the
libraries necessary to implement these tools.

Unless otherwise noted, none of these libraries have stable APIs.
Their main purpose is to aid the implementation of the tools.
You'll have to expect semiregular backwards-incompatible changes if you decide to use these libraries.

## System requirements

Releases support the current and previous version of Go at the time of release.
The master branch supports the current version of Go.
