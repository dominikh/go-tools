---
title: "Options"
description: "Explanations for all options"
aliases:
  - /docs/options
---

## checks {#checks}

This option sets which [checks]({{< relref "/docs/checks" >}}) should be enabled.
By default, most checks will be enabled, except for those that are too opinionated or that only apply to packages in certain domains.

All supported checks can be enabled with `"all"`.
Subsets of checks can be enabled via prefixes and the `*` glob; for example, `"S*"`, `"SA*"` and `"SA1*"` will
enable all checks in the S, SA and SA1 subgroups respectively.
Individual checks can be enabled by their full IDs.
To disable checks, prefix them with a minus sign. This works on all of the previously mentioned values.

Default value: `["all", "-{{< check "ST1000" >}}", "-{{< check "ST1003" >}}", "-{{< check "ST1016" >}}", "-{{< check "ST1020" >}}", "-{{< check "ST1021" >}}", "-{{< check "ST1022" >}}"]`

## initialisms {#initialisms}

{{< check "ST1003" >}} checks, among other
things, for the correct capitalization of initialisms. The
set of known initialisms can be configured with this option.

Default value: `["ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "QPS", "RAM", "RPC", "SLA", "SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "GID", "UID", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XMPP", "XSRF", "XSS", "SIP", "RTP", "AMQP", "DB", "TS"]`

## dot_import_whitelist {#dot_import_whitelist}

By default, {{< check "ST1001" >}} forbids
all uses of dot imports in non-test packages. This
setting allows setting a whitelist of import paths that can
be dot-imported anywhere.

Default value: `["github.com/mmcloughlin/avo/build", "github.com/mmcloughlin/avo/operand", "github.com/mmcloughlin/avo/reg"]`

## http_status_code_whitelist {#http_status_code_whitelist}

{{< check "ST1013" >}} recommends using constants from the `net/http` package
instead of hard-coding numeric HTTP status codes. This
setting specifies a list of numeric status codes that this
check does not complain about.

Default value: `["200", "400", "404", "500"]`
