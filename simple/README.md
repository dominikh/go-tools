Gosimple is a linter for Go source code that specialises on
simplifying code.

## Installation

Gosimple requires Go 1.6 or later.

    go get honnef.co/go/simple/cmd/gosimple

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

Gosimple will never contain rules that are also present in golint,
even if they would fit into gosimple. If golint should merge one of
gosimple's rules, it will be removed from gosimple shortly after, to
avoid duplicate results. It is strongly suggested that you use golint
and gosimple together and consider gosimple an addon to golint.

## Checks

Gosimple makes the following recommendations for avoiding unsimple
constructs:

- Don't use `select{}` with a single case. Instead, use a plain
  channel send or receive.
- Don't use `for { select {} }` with a single receive case. Instead,
  use `range` to iterate over the channel.
- Don't compare boolean expressions to the constants `true` or
  `false`. `if x == true` can be written as `if x` instead.
- Don't use `strings.Index*` or `bytes.Index` when you could use
  `strings.Contains*` and `bytes.Contains` instead.
- Don't use `bytes.Compare` to check for equality, use `bytes.Equal`.
- Don't use `for` loops to copy slices, use `copy`
- Don't use `for` loops to append one slice to another, use `x =
  append(x, y...)`
- Don't use `for _ = range x`, use `for range x`
- Don't use `for true { ... }`, use `for { ... }`
- Use raw strings with regexp.Compile to avoid two levels of escaping
- Don't use `if <expr> { return <bool> }; return <bool>`, use `return
  <expr>`, unless the `if` is one in a series of many early returns.
- Don't check if slices, maps or channels are nil before checking
  their length, it's redundant. `len` is defined as zero for those nil
  values.
- Don't use `time.Now().Sub(x)`, use `time.Since(x)` instead
- Don't write

  ```
  if err != nil {
    return err
  }
  return nil
  ```

  write

  ```
  return err
  ```

  instead
- Don't use `_ = <-ch`, use `<-ch` instead
- Use `strconv.Itoa` instead of `strconv.FormatInt` when it's simpler.

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
```
