---
"transports/axiom": major
"transports/blank": major
"transports/betterstack": major
"transports/charmlog": major
"transports/cli": major
"transports/console": major
"transports/datadog": major
"transports/gcplogging": major
"transports/http": major
"transports/logrus": major
"transports/lumberjack": major
"transports/newrelic": major
"transports/otellog": major
"transports/phuslu": major
"transports/pretty": major
"transports/sentry": major
"transports/slog": major
"transports/structured": major
"transports/testing": major
"transports/zap": major
"transports/zerolog": major
"integrations/loghttp": major
"integrations/sloghandler": major
"plugins/datadogtrace": major
"plugins/fmtlog": major
"plugins/oteltrace": major
"plugins/plugintest": major
"plugins/redact": major
"plugins/sampling": major
---

**Breaking: module paths bump to the next major.** These transports, plugins, and integrations now depend on `go.loglayer.dev/v3` and re-export v3 core types (or sibling v3 types) in their public API. Per Go convention, each module's path moves to its next major (`/v2` → `/v3`, or unversioned → `/v2`); consumers update imports and `go get` lines.

The `structured` transport also sanitizes ANSI escape sequences, bidi overrides, and CR/LF at the top level of every entry and omits the `msg` key when the message is empty. See the migration guide at https://go.loglayer.dev/migrating.
