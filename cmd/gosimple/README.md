# gosimple

_gosimple_ is a linter for Go source code that specialises on
simplifying code.

## Installation

Gosimple requires Go 1.6 or later.

    go get honnef.co/go/tools/cmd/gosimple

## Usage

Invoke `gosimple` with one or more filenames, a directory, or a package named
by its import path. Gosimple uses the same
[import path syntax](https://golang.org/cmd/go/#hdr-Import_path_syntax) as
the `go` command and therefore
also supports relative import paths like `./...`. Additionally the `...`
wildcard can be used as suffix on relative and absolute file paths to recurse
into them.

The output of this tool is a list of suggestions in Vim quickfix format,
which is accepted by lots of different editors.

## Purpose

Gosimple differs from golint in that gosimple focuses on simplifying
code, while golint flags common style issues. Furthermore, gosimple
always targets the latest Go version. If a new Go release adds a
simpler way of doing something, gosimple will suggest that way.

## Checks

Gosimple makes the following recommendations for avoiding unsimple
constructs:

| Check | Description                                                                 | Suggestion                                                               |
|-------|-----------------------------------------------------------------------------|--------------------------------------------------------------------------|
| S1000 | `select{}` with a single case                                               | Use a plain channel send or receive                                      |
| S1001 | A loop copying elements of `s2` to `s1`                                     | `copy(s1, s2)`                                                           |
| S1002 | `if b == true`                                                              | `if b`                                                                   |
| S1003 | `strings.Index*(x, y) != -1`                                                | `strings.Contains(x, y)`                                                 |
| S1004 | `bytes.Compare(x, y) == 0`                                                  | `bytes.Equal(x, y)`                                                      |
| S1005 | `for _ = range x`                                                           | `for range x`                                                            |
| S1006 | `for true {...}`                                                            | `for {...}`                                                              |
| S1007 | Using double quotes and escaping for regular expressions                    | Use raw strings                                                          |
| S1008 | `if <expr> { return <bool> }; return <bool>`                                | `return <expr>`                                                          |
| S1009 | Checking a slice against nil and also checking its length against zero      | Nil slices are defined to have length zero, the nil check is redundant   |
| S1010 | `s[a:len(s)]`                                                               | `s[a:]`                                                                  |
| S1011 | A loop appending each element of `s2` to `s1`                               | `append(s1, s2...)`                                                      |
| S1012 | `time.Now().Sub(x)`                                                         | `time.Since(x)`                                                          |
| S1013 | `if err != nil { return err }; return nil`                                  | `return err`                                                             |
| S1014 | `_ = <-x`                                                                   | `<-x`                                                                    |
| S1015 | Using `strconv.FormatInt` when `strconv.Atoi` would be more straightforward |                                                                          |
| S1016 | Converting two struct types by manually copying each field                  | A type conversion: `T(v)`                                                |
| S1017 | `if strings.HasPrefix` + string slicing                                     | Call `strings.TrimPrefix` unconditionally                                |
| S1018 | A loop sliding elements in a slice to the beginning                         | `copy(s[:n], s[offset:])`                                                |
| S1019 | `make(T, 0)` or `make(T, x, x)`                                             | `make(T)` or `make(T, x)`                                                |
| S1020 | `if _, ok := i.(T); ok && i != nil`                                         | `if _, ok := i.(T); ok`                                                  |
| S1021 | `var x uint; x = 1`                                                         | `var x uint = 1`                                                         |
| S1022 | `x, _ = someMap[key]`                                                       | `x = someMap[key]`                                                       |
| S1023 | `break` as the final statement of a `case` clause                           | Go doesn't have automatic fallthrough, making final `break` redundant    |
| S1024 | `t.Sub(time.Now())`                                                         | `time.Until(t)`                                                          |
| S1025 | `fmt.Sprintf("%s", x)` where `x` is already a string                        | `x`                                                                      |
|       | `fmt.Sprintf("%s", x)` where `x`'s underlying type is a string              | `string(x)`                                                              |
|       | `fmt.Sprintf("%s", x)` where `x` has a String method                        | `x.String()`                                                             |
| S1026 | Copies of strings, like `string([]byte(x))` or `"" + x`                     | `x`                                                                      |
| S1027 | `return` as the final statement of a func body with no return values        | Functions that don't return anything don't need a final return statement |
| S1028 | `errors.New(fmt.Sprintf(...))`                                              | `fmt.Errorf(...)`                                                        |
| S1029 | `for _, r := range []rune(s)`                                               | `for _, r := range s`                                                    |
| S1030 | `string(buf.Bytes())` or `[]byte(buf.String())`                           | Use the appropriate method of `bytes.Buffer` instead                     |

## gofmt -r

Some of these rules can be automatically applied via `gofmt -r`:

```
strings.IndexRune(a, b) > -1 -> strings.ContainsRune(a, b)
strings.IndexRune(a, b) >= 0 -> strings.ContainsRune(a, b)
strings.IndexRune(a, b) != -1 -> strings.ContainsRune(a, b)
strings.IndexRune(a, b) == -1 -> !strings.ContainsRune(a, b)
strings.IndexRune(a, b) < 0 -> !strings.ContainsRune(a, b)
strings.IndexAny(a, b) > -1 -> strings.ContainsAny(a, b)
strings.IndexAny(a, b) >= 0 -> strings.ContainsAny(a, b)
strings.IndexAny(a, b) != -1 -> strings.ContainsAny(a, b)
strings.IndexAny(a, b) == -1 -> !strings.ContainsAny(a, b)
strings.IndexAny(a, b) < 0 -> !strings.ContainsAny(a, b)
strings.Index(a, b) > -1 -> strings.Contains(a, b)
strings.Index(a, b) >= 0 -> strings.Contains(a, b)
strings.Index(a, b) != -1 -> strings.Contains(a, b)
strings.Index(a, b) == -1 -> !strings.Contains(a, b)
strings.Index(a, b) < 0 -> !strings.Contains(a, b)
bytes.Index(a, b) > -1 -> bytes.Contains(a, b)
bytes.Index(a, b) >= 0 -> bytes.Contains(a, b)
bytes.Index(a, b) != -1 -> bytes.Contains(a, b)
bytes.Index(a, b) == -1 -> !bytes.Contains(a, b)
bytes.Index(a, b) < 0 -> !bytes.Contains(a, b)
bytes.Compare(a, b) == 0 -> bytes.Equal(a, b)
bytes.Compare(a, b) != 0 -> !bytes.Equal(a, b)

time.Now().Sub(a) -> time.Since(a)
a.Sub(time.Now()) -> time.Until(a)
```

## Ignoring checks

gosimple allows disabling some or all checks for certain files. The
`-ignore` flag takes a whitespace-separated list of
`glob:check1,check2,...` pairs. `glob` is a glob pattern matching
files in packages, and `check1,check2,...` are checks named by their
IDs.

For example, to ignore uses of strconv.FormatInt in all test files in the
`os/exec` package, you would write `-ignore
"os/exec/*_test.go:S1015"`

Additionally, the check IDs support globbing, too. Using a pattern
such as `os/exec/*.gen.go:*` would disable all checks in all
auto-generated files in the os/exec package.

Any whitespace can be used to separate rules, including newlines. This
allows for a setup like the following:

```
$ cat stdlib.ignore
sync/*_test.go:S1000
testing/benchmark.go:S1016
runtime/string_test.go:S1005

$ gosimple -ignore "$(cat stdlib.ignore)" std
```
