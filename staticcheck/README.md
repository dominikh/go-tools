Staticcheck is a tool for statically check the input to certain
functions, such as `regexp.Compile`.


## Installation

Staticcheck requires Go 1.6 or later.

    go get honnef.co/go/staticcheck/cmd/staticcheck

## Usage

Invoke `staticcheck` with one or more filenames, a directory, or a package named
by its import path. Staticcheck uses the same
[import path syntax](https://golang.org/cmd/go/#hdr-Import_path_syntax) as
the `go` command and therefore
also supports relative import paths like `./...`. Additionally the `...`
wildcard can be used as suffix on relative and absolute file paths to recurse
into them.

The output of this tool is a list of suggestions in Vim quickfix format,
which is accepted by lots of different editors.

## Purpose

Staticcheck checks the input to functions like `regexp.Compile` and
time.Parse and makes sure that they conform to the API contract. It
does so by finding function calls with constant inputs and then
evaluating these inputs in the same way the code would at runtime.

For example, for `regexp.Compile("foo(")`, staticcheck will find the
call to `regexp.Compile` and check if `foo(` is a valid regexp.

The main purpose of staticcheck is editor integration, or workflow
integration in general. For example, by running staticcheck when
saving a file, one can quickly catch simple bugs without having to run
the whole test suite or the program itself.

The tool shouldn't report any errors unless there are legitimate bugs
in the code (or the tool…)

## Checks

The following function calls are currently checked by staticcheck:

- `regexp.Compile` and `regexp.MustCompile` - Checks that the regexp
  is valid
- `time.Parse` - Checks that the time format is valid
- `encoding/binary.Write` - Checks that the written value is supported
  by `encoding/binary`
- `text/template.Template.Parse` and `html/template.Template.Parse` –
  Check that the template is syntactically valid
