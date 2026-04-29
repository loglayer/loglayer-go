---
title: What's New
description: User-visible changes to LogLayer for Go.
---

# What's New

## v1.0.0 (unreleased)

Initial release. Stable API; SemVer applies from this point forward.

LogLayer for Go is a transport-agnostic structured logging facade. One fluent API on top of any logging library, JSON, terminal, HTTP, OpenTelemetry, or your own transport.

### What ships

- **Core API**: fluent builder with persistent fields, per-call metadata, errors, Go context, and group routing. Seven log levels (Trace through Panic). Every method safe from any goroutine. See [Getting Started](/getting-started).
- **Renderers**: [pretty](/transports/pretty), [structured](/transports/structured), [console](/transports/console), [testing](/transports/testing), [blank](/transports/blank).
- **Logger wrappers**: [zerolog](/transports/zerolog), [zap](/transports/zap), [log/slog](/transports/slog), [logrus](/transports/logrus), [charmbracelet/log](/transports/charmlog), [phuslu/log](/transports/phuslu).
- **Network transports**: [HTTP](/transports/http), [Datadog](/transports/datadog).
- **OpenTelemetry**: [`transports/otellog`](/transports/otellog) ships through the OTel logs pipeline; [`plugins/oteltrace`](/plugins/oteltrace) injects `trace_id`/`span_id` on any transport.
- **Plugins**: [redact](/plugins/redact), [sampling](/plugins/sampling), [fmtlog](/plugins/fmtlog), [datadogtrace](/plugins/datadogtrace), [oteltrace](/plugins/oteltrace). Six lifecycle hooks with centralized panic recovery; author your own per [Creating Plugins](/plugins/creating-plugins).
- **HTTP middleware**: [`integrations/loghttp`](/integrations/loghttp) binds a per-request logger to `r.Context()` and logs each response.
- **slog interop**: [`integrations/sloghandler`](/integrations/sloghandler) routes every `slog.Info(...)` call (yours and your dependencies') through the loglayer pipeline. Pairs with the [slog transport](/transports/slog) for the opposite direction.
- **stdlib log bridge**: `log.Writer(level)` / `log.NewLogLogger(level)` plumb anything that wants a `*log.Logger` or `io.Writer` (`http.Server.ErrorLog`, gorm, etc.) through loglayer.
- **Error chain expansion**: opt-in [`loglayer.UnwrappingErrorSerializer`](/logging-api/error-handling#built-in-unwrappingerrorserializer) surfaces every wrapped cause as a `causes` array.
- **Source / caller info**: opt-in via `Config.Source.Enabled`; the slog handler forwards `Record.PC` for free.
- **Runtime control**: live and concurrency-safe level changes, transport hot-swap, plugin add/remove, mute toggles.
- **Opt-in transport panic recovery**: `Config.OnTransportPanic` absorbs panics from `SendToLogger` so one bad sink can't suppress the others.
- **Defensive defaults**: CRLF / ANSI / Unicode-bidi sanitization in console and pretty, cross-host redirect refusal on the HTTP transport, redacted Datadog API key, cycle-safe `maputil.Cloner`.

For Go users coming from the TypeScript [`loglayer`](https://loglayer.dev) library, see [For TypeScript Developers](/for-typescript-developers) for the API mapping.
