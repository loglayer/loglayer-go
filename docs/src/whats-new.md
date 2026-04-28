---
title: What's New
description: User-visible changes to LogLayer for Go.
---

# What's New

## v1.0.0 (unreleased)

Initial release. Stable API; SemVer applies from this point forward.

LogLayer for Go is a transport-agnostic structured logging facade. One fluent API on top of any logging library, JSON, terminal, HTTP, OpenTelemetry, or your own transport.

### What ships

- **Core**: fluent builder API with persistent fields, per-call metadata, errors, Go context, and group-based routing. Distinct `Fields`/`Metadata`/`Data` types so the compiler catches misuse; `loglayer.F` and `loglayer.M` are short aliases for terser call sites. Every method safe from any goroutine. **Seven log levels** (Trace, Debug, Info, Warn, Error, Fatal, Panic). See [Getting Started](/getting-started).
- **Renderers**: [pretty](/transports/pretty) (colorized terminal, recommended for dev), [structured](/transports/structured) (JSON, recommended for production), [console](/transports/console), [testing](/transports/testing), [blank](/transports/blank).
- **Logger wrappers**: [zerolog](/transports/zerolog), [zap](/transports/zap), [log/slog](/transports/slog), [logrus](/transports/logrus), [charmbracelet/log](/transports/charmlog), [phuslu/log](/transports/phuslu). Drop LogLayer in front of an existing stack without rewriting your call sites.
- **Network**: [HTTP](/transports/http) (generic batched POST with pluggable encoder), [Datadog](/transports/datadog) (Logs HTTP intake with on-prem URL override).
- **OpenTelemetry**: [`transports/otellog`](/transports/otellog) ships entries through the OTel logs pipeline; [`plugins/oteltrace`](/plugins/oteltrace) injects `trace_id`/`span_id` on any transport. Both live in their own Go modules so the OTel SDK's dep graph doesn't bind users who don't import them.
- **Plugins**: six lifecycle hooks with centralized panic recovery. Built-in: [redact](/plugins/redact), [sampling](/plugins/sampling) (FixedRate / FixedRatePerLevel / Burst), [fmtlog](/plugins/fmtlog) (opt-in `fmt.Sprintf` semantics for multi-arg messages), [datadogtrace](/plugins/datadogtrace), [oteltrace](/plugins/oteltrace). Author your own per [Creating Plugins](/plugins/creating-plugins).
- **HTTP middleware**: [`integrations/loghttp`](/integrations/loghttp) derives a per-request logger, binds `r.Context()`, emits a "request completed" log on every response (or "request panicked" with the panic value if a handler crashes).
- **slog interop**: [`integrations/sloghandler`](/integrations/sloghandler) is a `log/slog.Handler` backed by a loglayer logger, so a single `slog.SetDefault(slog.New(sloghandler.New(log)))` line makes every `slog.Info(...)` call (yours and your dependencies') flow through loglayer's plugin pipeline, multi-transport fan-out, group routing, and level state. Pairs with the existing [slog Transport](/transports/slog) which covers the opposite direction (loglayer emitting through a `*slog.Logger` backend). Source info from `slog.Record.PC` is forwarded automatically.
- **stdlib log / `io.Writer` bridge**: `log.Writer(level)` returns an `io.Writer`; `log.NewLogLogger(level)` returns a `*log.Logger`. Drop into `http.Server.ErrorLog`, gorm, anything that wants a `*log.Logger` or `io.Writer`. See [Basic Logging → stdlib log and io.Writer Bridges](/logging-api/basic-logging#stdlib-log-and-io-writer-bridges).
- **Error chain expansion**: opt-in [`loglayer.UnwrappingErrorSerializer`](/logging-api/error-handling#built-in-unwrappingerrorserializer) walks `errors.Unwrap` and `errors.Join` and surfaces every wrapped cause as a `causes` array. Default serializer unchanged.
- **Source / caller info**: opt-in via `Config.Source.Enabled` (default off). When on, every emission captures file/line/function and surfaces them under `Config.Source.FieldName` (default `"source"`). JSON tags match the slog convention. Costs one `runtime.Caller` per emission when enabled. The slog handler forwards `Record.PC` for free, so source rendering on the slog path requires no toggle on the loglayer side. See [Configuration](/configuration#source-caller-info).
- **Runtime control**: change levels, hot-swap transports, add/remove plugins, mute fields/metadata, all live and concurrency-safe. Async-transport flushing on `Fatal` and on transport mutators is bounded by `Config.TransportCloseTimeout` (default 5 seconds) so a wedged endpoint can't hang the process or operator goroutine.
- **Transport panic recovery, opt-in.** Set `Config.OnTransportPanic` to recover panics from a transport's `SendToLogger` and report them via callback; the dispatch loop continues to the remaining transports so one bad sink doesn't suppress the others. Default (nil) keeps the hot path a direct call and lets panics propagate, matching the Go logging convention. The callback receives a `*RecoveredPanicError` (same shape that plugin hooks already use), so one observability function can absorb panics from either source. See [Configuration → OnTransportPanic](/configuration#ontransportpanic).
- **Defensive defaults**: console and pretty sanitize message strings against CRLF / ANSI / Unicode-bidi injection; `loghttp` sanitizes incoming HTTP headers; `maputil.Cloner` is cycle-safe; the Datadog config redacts its API key via `String()` and a `json:"-"` tag, and rejects non-https URLs (opt out with `AllowInsecureURL: true`) plus refuses silent overrides of `HTTP.URL` / `HTTP.Encoder` (`ErrHTTPOverrideForbidden`). The [redact plugin](/plugins/redact) walks the error subtree LogLayer builds from `WithError` so pattern-style redactors catch secrets baked into `err.Error()`. The [HTTP transport](/transports/http) refuses cross-host redirects on its default `Client` (so credential headers like `Authorization` / `X-API-Key` / `DD-API-KEY` can't leak to a redirected host) and recovers panics from user-supplied `Encoder` / `OnError` callbacks so a buggy callback can't silently halt log delivery.

For Go users coming from the TypeScript [`loglayer`](https://loglayer.dev) library, see [For TypeScript Developers](/for-typescript-developers) for the API mapping.

### Known gotchas

Each per-API page calls out its own traps inline: assign the result of `WithFields` / `WithCtx`; treat `Fields` and `Metadata` maps as read-only after handing them off; mute toggles are setup-time only; phuslu can't suppress its `os.Exit` on Fatal (every other wrapper honors `Config.DisableFatalExit`).
