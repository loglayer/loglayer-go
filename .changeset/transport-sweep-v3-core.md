---
"plugins/fmtlog": patch
"plugins/plugintest": patch
"plugins/redact": patch
"plugins/sampling": patch
"transports/betterstack": patch
"transports/datadog": patch
"transports/newrelic": patch
---

Upgrades the core dependency to `go.loglayer.dev/v3`. No user-visible behavior changes; release so the modules' dependency graphs resolve against v3.
