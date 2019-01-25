# honnef.co/go/tools

`honnef.co/go/tools/...` is a collection of tools and libraries for
working with Go code, including linters and static analysis, most
prominently staticcheck.

**These tools are supported by
[patrons on Patreon](https://www.patreon.com/dominikh) and
[sponsors](#sponsors). If you use these tools at your company,
consider supporting open source by [becoming a sponsor!](mailto:dominik@honnef.co?subject=Staticcheck%20sponsorship)**

## Installation

### Releases

It is recommended that you run released versions of the tools. These
releases can be found as git tags (e.g. `2019.1`) as well as prebuilt
binaries in the [releases tab](https://github.com/dominikh/go-tools/releases).

The easiest way of using the releases from source is to use a Go
package manager such as Godep or Go modules. Alternatively you can use
a combination of `git clone -b` and `go get` to check out the
appropriate tag and download its dependencies.


### Master

You can also run the master branch instead of a release. Note that
while the master branch is usually stable, it may still contain new
checks or backwards incompatible changes that break your build. By
using the master branch you agree to become a beta tester.

To use the master branch, a simple `go get -u
honnef.co/go/tools/cmd/...` suffices. You can also install a subset of
the commands, for example only staticcheck with `go get -u
honnef.co/go/tools/cmd/staticcheck`.

## Tools

All of the following tools can be found in the cmd/ directory. Each
tool is accompanied by its own README, describing it in more detail.

| Tool                                               | Description                                                             |
|----------------------------------------------------|-------------------------------------------------------------------------|
| [keyify](cmd/keyify/)                              | Transforms an unkeyed struct literal into a keyed one.                  |
| [rdeps](cmd/rdeps/)                                | Find all reverse dependencies of a set of packages                      |
| [staticcheck](cmd/staticcheck/)                    | Go static analysis, detecting bugs, performance issues, and much more. |
| [structlayout](cmd/structlayout/)                  | Displays the layout (field sizes and padding) of structs.               |
| [structlayout-optimize](cmd/structlayout-optimize) | Reorders struct fields to minimize the amount of padding.               |
| [structlayout-pretty](cmd/structlayout-pretty)     | Formats the output of structlayout with ASCII art.                      |

## Libraries

In addition to the aforementioned tools, this repository contains the
libraries necessary to implement these tools.

Unless otherwise noted, none of these libraries have stable APIs.
Their main purpose is to aid the implementation of the tools. If you
decide to use these libraries, please vendor them and expect regular
backwards-incompatible changes.

## System requirements

We support the last two versions of Go.

## Documentation

You can find extensive documentation on
[staticcheck.io](https://staticcheck.io).

## Sponsors

This project is sponsored by:

[<img src="images/sponsors/digitalocean.png" alt="DigitalOcean" height="35"></img>](https://digitalocean.com)  
[<img src="images/sponsors/fastly.png" alt="Fastly" height="55"></img>](https://fastly.com)  
[<img src="images/sponsors/uber.png" alt="Uber" height="35"></img>](https://uber.com)

## Licenses

All original code in this repository is licensed under the following
MIT license.

> Copyright (c) 2016 Dominik Honnef
>
> Permission is hereby granted, free of charge, to any person obtaining
> a copy of this software and associated documentation files (the
> "Software"), to deal in the Software without restriction, including
> without limitation the rights to use, copy, modify, merge, publish,
> distribute, sublicense, and/or sell copies of the Software, and to
> permit persons to whom the Software is furnished to do so, subject to
> the following conditions:
>
> The above copyright notice and this permission notice shall be
> included in all copies or substantial portions of the Software.
>
> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
> EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
> MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
> NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
> LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
> OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
> WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

In addition, some libraries reuse code owned by The Go Authors and
licensed under the following BSD 3-clause license:

> Copyright (c) 2013 The Go Authors. All rights reserved.
>
> Redistribution and use in source and binary forms, with or without
> modification, are permitted provided that the following conditions are
> met:
>
>    * Redistributions of source code must retain the above copyright
> notice, this list of conditions and the following disclaimer.
>    * Redistributions in binary form must reproduce the above
> copyright notice, this list of conditions and the following disclaimer
> in the documentation and/or other materials provided with the
> distribution.
>    * Neither the name of Google Inc. nor the names of its
> contributors may be used to endorse or promote products derived from
> this software without specific prior written permission.
>
> THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
> "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
> LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
> A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
> OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
> SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
> LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
> DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
> THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
> (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
> OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
