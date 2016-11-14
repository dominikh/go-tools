Staticcheck is `go vet` on steroids, applying a ton of static analysis
checks you might be used to from tools like ReSharper for C#.


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

The main purpose of staticcheck is editor integration, or workflow
integration in general. For example, by running staticcheck when
saving a file, one can quickly catch simple bugs without having to run
the whole test suite or the program itself.

The tool shouldn't report any errors unless there are legitimate
bugs - or very dubious constructs - in the code.

It is similar in nature to `go vet`, but has more checks that catch
bugs that would also be caught easily at runtime, to reduce the number
of edit, compile and debug cycles.

## Checks

The following things are currently checked by staticcheck:

| Check      | Description                                                                                                |
|------------|------------------------------------------------------------------------------------------------------------|
| **SA1???** | **Various misuses of the standard library**                                                                |
| SA1000     | Invalid regular expression                                                                                 |
| SA1001     | Invalid template                                                                                           |
| SA1002     | Invalid format in time.Parse                                                                               |
| SA1003     | Unsupported argument to functions in encoding/binary                                                       |
| SA1004     | Suspiciously small untyped constant in time.Sleep                                                          |
| SA1005     | Invalid first argument to exec.Command                                                                     |
| SA1006     | Printf with dynamic first argument and no further arguments                                                |
| SA1007     | Invalid URL in net/url.Parse                                                                               |
| SA1008     | Non-canonical key in http.Header map                                                                       |
| SA1009     | Incorrect usage of some standard library functions                                                         |
|            |                                                                                                            |
| **SA2???** | **Concurrency issues**                                                                                     |
| SA2000     | `sync.WaitGroup.Add` called inside the goroutine, leading to a race condition                              |
| SA2001     | Empty critical section, did you mean to `defer` the unlock?                                                |
| SA2002     | Called testing.T.FailNow or SkipNow in a goroutine, which isn't allowed                                    |
|            |                                                                                                            |
| **SA3???** | **Testing issues**                                                                                         |
| SA3000     | TestMain doesn't call os.Exit, hiding test failures                                                        |
| SA3001     | Assigning to `b.N` in benchmarks distorts the results                                                      |
|            |                                                                                                            |
| **SA4???** | **Code that isn't really doing anything**                                                                  |
| SA4000     | Boolean expression has identical expressions on both sides                                                 |
| SA4001     | `&*x` gets simplified to `x`, it does not copy `x`                                                         |
| SA4002     | Comparing strings with known different sizes has predictable results                                       |
| SA4003     | Comparing unsigned values against negative values is pointless                                             |
| SA4004     | The loop exits unconditionally after one iteration                                                         |
| SA4005     | Field assignment that will never be observed. Did you mean to use a pointer receiver?                      |
| SA4006     | A value assigned to a variable is never read before being overwritten. Forgotten error check or dead code? |
| SA4007     | Boolean expression always evaluates to the same result based on all known values of the operands           |
| SA4008     | The variable in the loop condition never changes, are you incrementing the wrong variable?                 |
| SA4009     | A function argument is overwritten before its first use                                                    |
| SA4010     | The result of `append` will never be observed anywhere                                                     |
| SA4011     | Break statement with no effect. Did you mean to break out of an outer loop?                                |
|            |                                                                                                            |
| **SA5???** | **Correctness issues**                                                                                     |
| SA5000     | Assignment to nil map                                                                                      |
| SA5001     | Defering `Close` before checking for a possible error                                                      |
| SA5002     | The empty `for` loop (`for {}`) spins and can block the scheduler                                          |
| SA5003     | Defers in infinite loops will never execute                                                                |
| SA5004     | `for { select { ...` with an empty default branch spins                                                    |
| SA5005     | The finalizer references the finalized object, preventing garbage collection                               |
| SA5006     | Slice index out of bounds                                                                                  |
|            |                                                                                                            |
| **SA9???** | **Dubious code constructs that have a high probability of being wrong**                                    |
| SA9000     | Storing non-pointer values in sync.Pool allocates memory                                                   |
| SA9001     | `defer`s in `for range` loops may not run when you expect them to                                          |
