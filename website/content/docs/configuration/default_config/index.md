---
headless: true
---
```toml
{{< option `checks` >}} = ["all", "-{{< check `ST1000` >}}", "-{{< check `ST1003` >}}", "-{{< check `ST1016` >}}", "-{{< check `ST1020` >}}", "-{{< check `ST1021` >}}", "-{{< check `ST1022` >}}", "-{{< check `ST1023` >}}"]
{{< option `initialisms` >}} = ["ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "QPS", "RAM", "RPC", "SLA", "SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "GID", "UID", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XMPP", "XSRF", "XSS", "SIP", "RTP", "AMQP", "DB", "TS"]
{{< option `dot_import_whitelist` >}} = ["github.com/mmcloughlin/avo/build", "github.com/mmcloughlin/avo/operand", "github.com/mmcloughlin/avo/reg"]
{{< option `http_status_code_whitelist` >}} = ["200", "400", "404", "500"]
```
