---
"transports/axiom": major
"transports/blank": major
"transports/charmlog": major
"transports/cli": major
"transports/console": major
"transports/gcplogging": major
"transports/http": major
"transports/logrus": major
"transports/lumberjack": major
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
"plugins/oteltrace": major
---

**Breaking: module paths bump to `/v3`.** These transports, plugins, and integrations now depend on `go.loglayer.dev/v3` and re-export v3 core types. Module paths move from `go.loglayer.dev/<path>/v2` to `go.loglayer.dev/<path>/v3`; update imports and `go get` lines.

The `structured` transport also sanitizes ANSI escape sequences, bidi overrides, and CR/LF at the top level of every entry and omits the `msg` key when the message is empty. See the migration guide at https://go.loglayer.dev/migrating.
