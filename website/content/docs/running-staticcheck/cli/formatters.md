---
title: "Formatters"
description: "Format Staticcheck's output in different ways"
aliases:
  - /docs/formatters
---

## Text {#text}

_Text_ is the default output formatter.
It formats problems using the following format: `file:line:col: message`.
This format is commonly used by compilers and linters,
and is understood by most editors.

### Example output
```text
go/src/fmt/print.go:1069:15: this value of afterIndex is never used (SA4006)
```

## Stylish {#stylish}

_Stylish_ is a formatter designed for human consumption.
It groups results by file name
and breaks up the various pieces of information into columns.
Additionally, it displays a final summary.

This output format is not suited for automatic consumption by tools
and may change between versions.

```text
go/src/fmt/fmt_test.go
(43, 2)     S1021   should merge variable declaration with assignment on next line
(1185, 10)  SA9003  empty branch

go/src/fmt/print.go
(77, 18)    ST1006  methods on the same type should have the same receiver name (seen 3x "b", 1x "bp")
(1069, 15)  SA4006  this value of afterIndex is never used

go/src/fmt/scan.go
(465, 5)  ST1012  error var complexError should have name of the form errFoo
(466, 5)  ST1012  error var boolError should have name of the form errFoo

✖ 6 problems (6 errors, 0 warnings)
```

## JSON {#json}

The JSON formatter emits one JSON object per problem found –
that is, it is a stream of objects, not an array.
Most fields should be self-explanatory.

The `severity` field may be one of
`"error"`, `"warning"` or `"ignored"`.
Whether a problem is an error or a warning is determined by the `-fail` flag.
The value `"ignored"` is used for problems that were ignored,
if the `-show-ignored` flag was provided.

### Example output

Note that actual output is not formatted nicely.
The example has been formatted to improve readability.

```json
{
  "code": "SA4006",
  "severity": "error",
  "location": {
    "file": "/usr/lib/go/src/fmt/print.go",
    "line": 1082,
    "column": 15
  },
  "end": {
    "file": "/usr/lib/go/src/fmt/print.go",
    "line": 1082,
    "column": 25
  },
  "message": "this value of afterIndex is never used"
}
```
