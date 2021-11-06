---
title: "Configuration"
description: "Tweak Staticcheck to your requirements"
weight: 3
---

Staticcheck tries to provide a good out-of-the-box experience, but it also offers a number of options to fine-tune it to your specific needs.

## Command-line flags {#cli-flags}

Staticcheck uses command-line flags for settings that are specific to single invocations and that may change depending on the context (local development, continuous integration etc).
This includes which build tags to set and which output format to use.
All of the CLI flags are explained in the [Command-line interface]({{< relref "/docs/running-staticcheck/cli" >}}) article.

## Configuration files {#configuration-files}

Staticcheck uses configuration files for settings that apply to all users of Staticcheck on a given project.
Configuration files can choose which checks to run as well as tweak the behavior of individual checks.

Configuration files are named `staticcheck.conf` and apply to subtrees of packages. Consider the following tree of Go packages and configuration files:

```plain
.
├── net
│   ├── cgi
│   ├── http
│   │   ├── parser
│   │   └── staticcheck.conf // config 3
│   └── staticcheck.conf     // config 2
├── staticcheck.conf         // config 1
└── strconv
```

Config 1 will apply to all packages, config 2 will apply to `./net/...` and config 3 will apply to `./net/http/...`.
When multiple configuration files apply to a package (for example, all three configs will apply to `./net/http`) they will be merged, with settings in files deeper in the package tree overriding rules higher up the tree.

### Configuration format {#configuration-format}

Staticcheck configuration files are named `staticcheck.conf` and contain [TOML](https://github.com/toml-lang/toml).

Any set option will override the same option from further up the package tree,
whereas unset options will inherit their values.
Additionally, the special value `"inherit"` can be used to inherit values.
This is especially useful for array values, as it allows adding and removing values to the inherited option.
For example, the option `checks = ["inherit", "ST1000"]` will inherit the enabled checks and additionally enable ST1000.

The special value `"all"` matches all possible values.
this is used when enabling or disabling checks.

Values prefixed with a minus sign,  such as `"-S1000"`  will exclude values from a list.
This can be used in combination with `"all"` to express "all but",
or in combination with `"inherit"` to remove values from the inherited option.

### Configuration options {#configuration-options}

A list of all options and their explanations can be found on the [Options]({{< relref "/docs/configuration/options" >}}) page.

### Example configuration {#example-configuration}

The following example configuration is the textual representation of Staticcheck's default configuration.

```toml
{{< option "checks" >}} = ["all", "-{{< check "ST1000" >}}", "-{{< check "ST1003" >}}", "-{{< check "ST1016" >}}", "-{{< check "ST1020" >}}", "-{{< check "ST1021" >}}", "-{{< check "ST1022" >}}", "-{{< check "ST1023" >}}"]
{{< option "initialisms" >}} = ["ACL", "API", "ASCII", "CPU", "CSS", "DNS",
	"EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID",
	"IP", "JSON", "QPS", "RAM", "RPC", "SLA",
	"SMTP", "SQL", "SSH", "TCP", "TLS", "TTL",
	"UDP", "UI", "GID", "UID", "UUID", "URI",
	"URL", "UTF8", "VM", "XML", "XMPP", "XSRF",
	"XSS", "SIP", "RTP", "AMQP", "DB", "TS"]
{{< option "dot_import_whitelist" >}} = [
    "github.com/mmcloughlin/avo/build",
    "github.com/mmcloughlin/avo/operand",
    "github.com/mmcloughlin/avo/reg",
]
{{< option "http_status_code_whitelist" >}} = ["200", "400", "404", "500"]
```


## Ignoring problems with linter directives {#ignoring-problems}

In general, you shouldn't have to ignore problems reported by Staticcheck.
Great care is taken to minimize the number of false positives and subjective suggestions.
Dubious code should be rewritten and genuine false positives should be reported so that they can be fixed.

The reality of things, however, is that not all corner cases can be taken into consideration.
Sometimes code just has to look weird enough to confuse tools,
and sometimes suggestions, though well-meant, just aren't applicable.
For those rare cases, there are several ways of ignoring unwanted problems.

### Line-based linter directives {#line-based-linter-directives}

The most fine-grained way of ignoring reported problems is to annotate the offending lines of code with linter directives.

The `//lint:ignore Check1[,Check2,...,CheckN] reason` directive
ignores one or more checks on the following line of code.
The `reason` is a required field
that must describe why the checks should be ignored for that line of code.
This field acts as documentation for other people (including future you) reading the code.

Let's consider the following example,
which intentionally checks that the results of two identical function calls are not equal:

```go
func TestNewEqual(t *testing.T) {
  if errors.New("abc") == errors.New("abc") {
    t.Errorf(`New("abc") == New("abc")`)
  }
}
```

{{< check "SA4000" >}} will flag this code,
pointing out that the left and right side of `==` are identical –
usually indicative of a typo and a bug.

To silence this problem, we can use a linter directive:

```go
func TestNewEqual(t *testing.T) {
  //lint:ignore SA4000 we want to make sure that no two results of errors.New are ever the same
  if errors.New("abc") == errors.New("abc") {
    t.Errorf(`New("abc") == New("abc")`)
  }
}
```

### Maintenance of linter directives {#maintenance-of-linter-directives}

It is crucial to update or remove outdated linter directives when code has been changed.
Staticcheck helps you with this by making unnecessary directives a problem of its own.
For example, for this (admittedly contrived) snippet of code

```go
//lint:ignore SA1000 we love invalid regular expressions!
regexp.Compile(".+")
```

Staticcheck will report the following:

```plain
tmp.go:1:2: this linter directive didn't match anything; should it be removed?
```

Checks that have been disabled via configuration files will not cause directives to be considered unnecessary.

### File-based linter directives {#file-based-linter-directives}

In some cases, you may want to disable checks for an entire file.
For example, code generation may leave behind a lot of unused code,
as it simplifies the generation process.
Instead of manually annotating every instance of unused code,
the code generator can inject a single, file-wide ignore directive to ignore the problem.

File-based linter directives look a lot like line-based ones:

```go
//lint:file-ignore U1000 Ignore all unused code, it's generated
```

The only difference is that these comments aren't associated with any specific line of code.
Conventionally, these comments should be placed near the top of the file.

Unlike line-based directives, file-based ones will not be flagged for being unnecessary.
