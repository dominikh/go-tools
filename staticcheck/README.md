Staticcheck is a tool for statically checking the inputs to and uses
of certain functions and constructs, such as `regexp.Compile`.


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

Furthermore, it checks that functions and constructs are used in the
correct way. It checks, for example, that sync.WaitGroup.Add is called
outside of a goroutine, to avoid race conditions.

The main purpose of staticcheck is editor integration, or workflow
integration in general. For example, by running staticcheck when
saving a file, one can quickly catch simple bugs without having to run
the whole test suite or the program itself.

The tool shouldn't report any errors unless there are legitimate bugs
in the code (or the tool…)

It is similar in nature to `go vet`, but differs in that staticcheck
has tests for mistakes that happen during development but wouldn't
make it into the final code, either due to tests, crashes or the race
detector. It is meant to reduce the number of edit, compile and debug
cycles by providing more instant feedback.

## Checks

The following things are currently checked by staticcheck:

- `regexp.Compile` and `regexp.MustCompile` - Checks that the regexp
  is valid
- `time.Parse` - Checks that the time format is valid
- `encoding/binary.Write` - Checks that the written value is supported
  by `encoding/binary`
- `text/template.Template.Parse` and `html/template.Template.Parse` –
  Check that the template is syntactically valid
- `time.Sleep` - Checks that the call doesn't use suspiciously small
  (<120), untyped literals. This usually indicates a bug, where
  `time.Sleep(1)` is assumed to sleep for 1 second, while in reality
  it sleeps for 1 nanosecond.
- `sync.WaitGroup.Add` – Checks that the method is called before
  launching the goroutine, to avoid a race condition.
- `sync.WaitGroup` - Checks that wait groups aren't copied when being
  passed to functions.
- `TestMain` - checks that the TestMain function calls os.Exit to
  report test failure. -- This check may have false positive. Its
  confidence is 0.9 and can be filtered with the `-min_confidence`
  flag set to `1`.
- `exec.Command` - checks that the first argument looks valid
- Dont use an empty `for {}` as it will spin.
- Don't use `defer` in a loop that will never finish.

## Examples

```
github.com/couchbase/indexing/secondary/tools/logd/main.go:134:33: error parsing regexp: missing closing ): ` FEED\[<=>([^\(]*)(`
github.com/jvehent/mig/client/mig-console/action_launcher.go:312:35: parsing time "2014-01-01T00:00:00.0-": month out of range
github.com/khlieng/dispatch/vendor/github.com/xenolf/lego/acme/crypto.go:165:42: type int cannot be used with binary.Write
github.com/netrack/openflow/ofp.v13/error.go:246:43: type *ofp.ErrorMsg cannot be used with binary.Write
github.com/sangelone/wigglr/benchmark_test.go:44:42: type *main.Wiggle cannot be used with binary.Write
```
