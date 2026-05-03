---
"go.loglayer.dev": patch
"integrations/loghttp": patch
"integrations/sloghandler": patch
"plugins/datadogtrace": patch
"plugins/fmtlog": patch
"plugins/oteltrace": patch
"plugins/plugintest": patch
"plugins/redact": patch
"plugins/sampling": patch
"transports/blank": patch
"transports/charmlog": patch
"transports/cli": patch
"transports/console": patch
"transports/datadog": patch
"transports/gcplogging": patch
"transports/http": patch
"transports/logrus": patch
"transports/lumberjack": patch
"transports/otellog": patch
"transports/phuslu": patch
"transports/pretty": patch
"transports/sentry": patch
"transports/slog": patch
"transports/structured": patch
"transports/testing": patch
"transports/zap": patch
"transports/zerolog": patch
---

Republish every module to ship a clean `go.mod` to the Go module proxy.

The v2.0.0 cascade and the subsequent `transports/cli` v2.1.0 release shipped sub-module `go.mod` files containing dev-only `replace go.loglayer.dev/v2 => ../..` directives and placeholder pseudo-version requires (`v2.0.0-00010101000000-000000000000`). Downstream consumers who depended on any sub-module saw `go mod tidy` 404 on the placeholder.

monorel v0.9.0 ([disaresta-org/monorel#42](https://github.com/disaresta-org/monorel/pull/42)) added a release-time `go.mod` cleaner that strips the dev-only sibling replaces and pins sibling requires to the planned release version. This release republishes every affected module with the cleaned `go.mod`.

No API changes. Re-`go get` to pick up the cleaned modules.
