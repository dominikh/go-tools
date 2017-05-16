# staticcheck

_staticcheck_ is `go vet` on steroids, applying a ton of static analysis
checks you might be used to from tools like ReSharper for C#.

## Installation

Staticcheck requires Go 1.6 or later.

    go get honnef.co/go/tools/cmd/staticcheck

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

|Check|Description|
|---|---|
|**SA1???**|**Various misuses of the standard library**|
|SA1000|Invalid regular expression|
|SA1001|Invalid template|
|SA1002|Invalid format in time.Parse|
|SA1003|Unsupported argument to functions in encoding/binary|
|SA1004|Suspiciously small untyped constant in time.Sleep|
|[SA1005](#SA1005)|Invalid first argument to exec.Command|
|[SA1006](#SA1006)|Printf with dynamic first argument and no further arguments|
|SA1007|Invalid URL in net/url.Parse|
|SA1008|Non-canonical key in http.Header map|
|SA1010|`(*regexp.Regexp).FindAll` called with n == 0, which will always return zero results|
|SA1011|Various methods in the `strings` package expect valid UTF-8, but invalid input is provided|
|SA1012|A nil `context.Context` is being passed to a function, consider using context.TODO instead|
|SA1013|`io.Seeker.Seek` is being called with the `whence` constant as the first argument, but it should be the second|
|SA1014|Non-pointer value passed to Unmarshal or Decode|
|SA1015|Using `time.Tick` in a way that will leak. Consider using `time.NewTicker`, and only use `time.Tick` in tests, commands and endless functions|
|SA1016|Trapping a signal that cannot be trapped|
|SA1017|Channels used with signal.Notify should be buffered|
|SA1018|`strings.Replace` called with n == 0, which does nothing|
|SA1019|Using a deprecated function, variable, constant or field|
|SA1020|Using an invalid `host:port` pair with a `net.Listen`-related function|
|[SA1021](#SA1021)|Using bytes.Equal to compare two net.IP|
|[SA1022](#SA1022)|Calling os.Exit in a function assigned to flag.Usage|
|SA1023|Modifying the buffer in an io.Writer implementation|
|SA1024|A string cutset contains duplicate characters, suggesting TrimPrefix or TrimSuffix should be used instead of TrimLeft or TrimRight|
|||
|**SA2???**|**Concurrency issues**|
|SA2000|`sync.WaitGroup.Add` called inside the goroutine, leading to a race condition|
|SA2001|Empty critical section, did you mean to `defer` the unlock?|
|SA2002|Called testing.T.FailNow or SkipNow in a goroutine, which isn't allowed|
|SA2003|Deferred Lock right after locking, likely meant to defer Unlock instead|
|||
|**SA3???**|**Testing issues**|
|SA3000|TestMain doesn't call os.Exit, hiding test failures|
|SA3001|Assigning to `b.N` in benchmarks distorts the results|
|||
|**SA4???**|**Code that isn't really doing anything**|
|SA4000|Boolean expression has identical expressions on both sides|
|SA4001|`&*x` gets simplified to `x`, it does not copy `x`|
|SA4002|Comparing strings with known different sizes has predictable results|
|SA4003|Comparing unsigned values against negative values is pointless|
|SA4004|The loop exits unconditionally after one iteration|
|SA4005|Field assignment that will never be observed. Did you mean to use a pointer receiver?|
|SA4006|A value assigned to a variable is never read before being overwritten. Forgotten error check or dead code?|
|SA4008|The variable in the loop condition never changes, are you incrementing the wrong variable?|
|SA4009|A function argument is overwritten before its first use|
|SA4010|The result of `append` will never be observed anywhere|
|SA4011|Break statement with no effect. Did you mean to break out of an outer loop?|
|SA4012|Comparing a value against NaN even though no value is equal to NaN|
|SA4013|Negating a boolean twice (`!!b`) is the same as writing `b`. This is either redundant, or a typo.|
|SA4014|An if/else if chain has repeated conditions and no side-effects; if the condition didn't match the first time, it won't match the second time, either|
|SA4015|Calling functions like math.Ceil on floats converted from integers doesn't do anything useful|
|SA4016|Certain bitwise operations, such as `x ^ 0`, do not do anything useful|
|SA4017|A pure function's return value is discarded, making the call pointless|
|||
|**SA5???**|**Correctness issues**|
|SA5000|Assignment to nil map|
|SA5001|Defering `Close` before checking for a possible error|
|SA5002|The empty `for` loop (`for {}`) spins and can block the scheduler|
|SA5003|Defers in infinite loops will never execute|
|SA5004|`for { select { ...` with an empty default branch spins|
|[SA5005](#SA5005)|The finalizer references the finalized object, preventing garbage collection|
|SA5006|Slice index out of bounds|
|[SA5007](#SA5007)|Infinite recursive call|
|||
|**SA6???**|**Performance issues**|
|SA6000|Using `regexp.Match` or related in a loop, should use `regexp.Compile`|
|[SA6001](#SA6001)|Missing an optimization opportunity when indexing maps by byte slices|
|[SA6002](#SA6002)|Storing non-pointer values in sync.Pool allocates memory|
|[SA6003](#SA6003)|Converting a string to a slice of runes before ranging over it|
|||
|**SA9???**|**Dubious code constructs that have a high probability of being wrong**|
|SA9001|`defer`s in `for range` loops may not run when you expect them to|
|SA9002|Using a non-octal `os.FileMode`  that looks like it was meant to be in octal.|
|SA9003|Empty body in an if or else branch|
|||

### <a id="SA1005">SA1005 – Invalid first argument to exec.Command

`os/exec` runs programs directly (using variants of the
[fork](https://en.wikipedia.org/wiki/Fork_(system_call)) and
[exec](https://en.wikipedia.org/wiki/Exec_(system_call)) system calls
on Unix systems). This shouldn't be confused with running a command in
a shell. The shell will allow for features such as input redirection,
pipes, and general scripting. The
shell is also responsible for splitting the user's input into a
program name and its arguments. For example, the equivalent to `ls /
/tmp` would be `exec.Command("ls", "/", "/tmp")`.

If you want to run a command in a shell, consider using something like
the following – but be aware that not all systems, particularly
Windows, will have a `/bin/sh` program:

```
exec.Command("/bin/sh", "-c", "ls | grep Awesome")
```
### <a id="SA1006">SA1006 – Printf with dynamic first argument and no further arguments

Using `fmt.Printf` with a dynamic first argument can lead to
unexpected output. The first argument is a format string, where
certain character combinations have special meaning. If, for example,
a user were to enter a string such as `Interest rate: 5%` and you
printed it with `fmt.Printf(s)`, it would lead to the following
output: `Interest rate: 5%!(NOVERB)`.

Similarly, forming the first parameyer via string concatenation with
user input should be avoided for the same reason. When printing user
input, either use a variant of `fmt.Print`, or use the `%s` Printf
verb and pass the string as an argument.
### <a id="SA1021">SA1021 – Using bytes.Equal to compare two net.IP

A `net.IP` stores an IPv4 or IPv6 address as a slice of bytes. The
length of the slice for an IPv4 address, however, can be either 4 or
16 bytes long, using different ways of representing IPv4 addresses. In
order to correctly compare two `net.IP`s, the `net.IP.Equal` method
should be used, as it takes both representations into account.
### <a id="SA1022">SA1022 – Calling os.Exit in a function assigned to flag.Usage

The `flag` package has the notion of a `Usage` function, assigned to
`flag.Usage` or `flag.FlagSet.Usage`. The job of this function is to
print usage instructions for the program and it is called when invalid
flags were provided.

This function should not, however, terminate the program by calling
`os.Exit`. The `flag` package already has a mechanism for exiting on
incorrect flags, the `errorHandling` argument of `flag.NewFlagSet`.
Setting it to `flag.ExitOnError` instructs it to call `os.Exit(2)`.
There exist other values to react differently, which is why `Usage`
shouldn't call `os.Exit` on its own.
### <a id="SA5005">SA5005 – The finalizer references the finalized object, preventing garbage collection

A finalizer is a function associated with an object that runs when the
garbage collector is ready to collect said object, that is when the
object is no longer referenced by anything.

If the finalizer references the object, however, it will always remain
as the final reference to that object, preventing the garbage
collector from collecting the object. The finalizer will never run,
and the object will never be collected, leading to a memory leak. That
is why the finalizer should instead use its first argument to operate
on the object. That way, the number of references can temporarily go
to zero before the object is being passed to the finalizer.
### <a id="SA5007">SA5007 – Infinite recursive call

A function that calls itself recursively needs to have an exit
condition. Otherwise it will recurse forever, until the system runs
out of memory.

This issue can be caused by simple bugs such as forgetting adding an
exit condition. It can also happen "on purpose". Some languages have
[tail call optimization](https://en.wikipedia.org/wiki/Tail_call)
which makes certain infinite recursive calls safe to use. Go, however,
does not implement TCO, and as such a loop should be used instead.
### <a id="SA6001">SA6001 – Missing an optimization opportunity when indexing maps by byte slices

Map keys must be comparable, which precludes the use of []byte. This
usually leads to using string keys and converting []bytes to
strings.

Normally, a conversion of []byte to string needs to copy the data and
causes allocations. The compiler, however, recognizes `m[string(b)]`
and uses the data of `b` directly, without copying it, because it
knows that the data can't change during the map lookup. This leads
to the counter-intuitive situation that

```
k := string(b)
println(m[k])
println(m[k])
```

will be less efficient than

```
println(m[string(b)])
println(m[string(b)])
```

because the first version needs to copy and allocate, while the second
one does not.

For some history on this optimization, check out commit
[f5f5a8b6209f84961687d993b93ea0d397f5d5bf](https://github.com/golang/go/commit/f5f5a8b6209f84961687d993b93ea0d397f5d5bf).
### <a id="SA6002">SA6002 – Storing non-pointer values in sync.Pool allocates memory

A `sync.Pool` is used to avoid unnecessary allocations and reduce the
amount of work the garbage collector has to do.

When passing a value that is larger than a single word (8 bytes on a
64 bit machine) to a function that accepts an interface, the value
needs to be placed on the heap, which means an additional allocation.
Slices are a common thing to put in `sync.Pool`s, and they're 3 words
large (length, capacity, and a pointer to an array). In order to avoid
the extra allocation, one should store a pointer to the slice instead.

See the
[comments on a Go CL](https://go-review.googlesource.com/#/c/24371/)
that discuss this problem.
### <a id="SA6003">SA6003 – Converting a string to a slice of runes before ranging over it

You may want to loop over the runes in a string. Instead of converting
the string to a slice of runes and looping over that, you can loop
over the string itself. That is,

```
for _, r := range s {}
```

and

```
for _, r := range []rune(s) {}
```

will yield the same values. The first version, however, will be faster
and avoid unnecessary memory allocations.

Do note that if you are interested in the indices, ranging over a
string and over a slice of runes will yield different indices. The
first one yields byte offsets, while the second one yields indices in
the slice of runes.

## Ignoring checks

staticcheck allows disabling some or all checks for certain files. The
`-ignore` flag takes a whitespace-separated list of
`glob:check1,check2,...` pairs. `glob` is a glob pattern matching
files in packages, and `check1,check2,...` are checks named by their
IDs.

For example, to ignore assignment to nil maps in all test files in the
`os/exec` package, you would write `-ignore
"os/exec/*_test.go:SA5000"`

Additionally, the check IDs support globbing, too. Using a pattern
such as `os/exec/*.gen.go:*` would disable all checks in all
auto-generated files in the os/exec package.

Any whitespace can be used to separate rules, including newlines. This
allows for a setup like the following:

```
$ cat stdlib.ignore
sync/*_test.go:SA2001
testing/benchmark.go:SA3001
runtime/string_test.go:SA4007
runtime/proc_test.go:SA5004
runtime/lfstack_test.go:SA4010
runtime/append_test.go:SA4010
errors/errors_test.go:SA4000
reflect/all_test.go:SA4000

$ staticcheck -ignore "$(cat stdlib.ignore)" std
```
