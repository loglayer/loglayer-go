---
title: What's New
description: User-visible changes to LogLayer for Go.
---

# What's New

Changes are recorded here once they ship. Until v0.1.0 lands, this page describes the API as it stands today.

## v0.1.0 (unreleased)

Initial release of LogLayer for Go: a transport-agnostic structured logging facade with a fluent API for messages, fields, metadata, and errors.

### Core

- Six log levels: Trace, Debug, Info, Warn, Error, Fatal.
- Fluent builder API: `WithMetadata`, `WithError`, `WithCtx`, chained into a level method (`Info`, `Warn`, etc.).
- `loglayer.Fields` (a `map[string]any` alias) for persistent, keyed data attached to a logger via `WithFields`. Returns a new logger on each call, matching the convention used by zerolog, zap, slog, and logrus.
- `WithoutFields(keys ...string)` returns a new logger with those keys removed (or all fields when called with no arguments).
- `loglayer.Metadata` (a `map[string]any` alias) for the common shape passed to `WithMetadata`. `WithMetadata` accepts any value; transports decide how to serialize structs and other types.
- `Child()` and `WithPrefix(prefix)` for explicit cloning.
- `Raw(RawLogEntry)` for bypassing the builder when forwarding pre-assembled entries from another system.
- `MetadataOnly` and `ErrorOnly` shortcuts.

### Constructors

- `loglayer.New(Config) *LogLayer` panics on misconfiguration; `loglayer.Build(Config) (*LogLayer, error)` returns `loglayer.ErrNoTransport` instead.
- `Config.DisableFatalExit` opts out of the default `os.Exit(1)` after a Fatal log (Go convention).
- `loglayer.NewMock()` returns a silent `*LogLayer` for tests, with `DisableFatalExit` enabled automatically.
- `Config.Disabled bool` (replaces the prior `*bool` `Enabled`). Same change for `transport.BaseConfig.Disabled`. The negated naming makes the zero-value default ("logger on") explicit.
- Setting both `Config.Transport` and `Config.Transports` is now rejected with `ErrTransportAndTransports`. Previously `Transport` silently won over `Transports`.
- `MetadataOnly(v, opts ...MetadataOnlyOpts)` now mirrors `ErrorOnly(err, opts ...ErrorOnlyOpts)`. Old: `log.MetadataOnly(m, loglayer.LogLevelWarn)`. New: `log.MetadataOnly(m, loglayer.MetadataOnlyOpts{LogLevel: loglayer.LogLevelWarn})`.
- `transports/console.ConsoleTransport` and `transports/structured.StructuredTransport` renamed to `Transport` (no more `console.ConsoleTransport` stutter). Other transports already used the unprefixed name.
- New `ErrPluginNoID` sentinel. `Build` returns it instead of panicking when a `Config.Plugins` entry has an empty ID. `New` and `AddPlugin` still panic, with the same sentinel.
- `transports/http` and `transports/datadog` gain `Build(Config) (*Transport, error)` siblings. New sentinels: `httptransport.ErrURLRequired` and `datadog.ErrAPIKeyRequired`. Use `Build` when the required URL or API key is loaded at runtime (e.g. from an environment variable).
- `TransportParams.HasData` removed (was always equal to `Data != nil`). Replace `if params.HasData` with `if len(params.Data) > 0`. Same change for `transports/testing.LogLine`.
- `integrations/loghttp.Middleware` now takes `loghttp.Config` instead of variadic functional options, matching every other package in the library. The `WithXxx` Option constructors are removed. Old: `loghttp.Middleware(log, loghttp.WithStartLog(true))`. New: `loghttp.Middleware(log, loghttp.Config{StartLog: true})`.
- `transports/zerolog.Config.DisableFatalExit` removed (was declared but never read). The "transport always honors loglayer's exit contract" rationale moved into the package doc comment.
- `ErrorOnlyOpts.CopyMsg` is now a typed `CopyMsgPolicy` enum instead of `*bool`. Values: `CopyMsgDefault` (zero, uses config), `CopyMsgEnabled`, `CopyMsgDisabled`. No more `&true` / `&false` ceremony.
- `WithFreshTransports` renamed to `SetTransports`. The previous name implied "returns new logger" via the `With*` prefix; the method actually mutates in place. `SetTransports` parallels `SetLevel`.
- `AddPlugin` is now variadic to match `AddTransport`: `log.AddPlugin(p1, p2, p3)` or `log.AddPlugin(plugins...)`.
- `transports/http` errors propagated to `OnError` now include the `loglayer/transports/http:` package prefix so callers can identify the source at a glance.
- `ClearFields` renamed to `WithoutFields`. The method returns a new logger; the prior `Clear*` prefix violated the "With* returns new" convention. Matches the TypeScript loglayer's `withoutContext` precedent.

### Groups

- Group routing ported from TS loglayer's `withGroup` feature. Define named routing rules in `Config.Groups`, then tag entries via `(*LogBuilder).WithGroup(...string)` (single entry) or `(*LogLayer).WithGroup(...string)` (returns a child where every log is tagged). Each group lists transport IDs to route to, plus optional minimum level and `Disabled` toggle. Backward compatible: when `Groups` is nil/empty, every transport receives every entry as before.
- `Config.ActiveGroups []string` filters routing to named groups; `Config.UngroupedRouting` (typed enum: `UngroupedToAll` / `UngroupedToNone` / `UngroupedToTransports`) controls untagged entries.
- Runtime mutators: `AddGroup`, `RemoveGroup`, `EnableGroup`, `DisableGroup`, `SetGroupLevel`, `SetActiveGroups`, `ClearActiveGroups`, `GetGroups`.
- `loglayer.ActiveGroupsFromEnv("LOGLAYER_GROUPS")` parses a comma-separated env-var list into `[]string` for `Config.ActiveGroups`. (We don't read environment variables on your behalf, but the helper makes the common case one line.)

### Shared utilities

- New `loglayer.MetadataPlugin`, `loglayer.FieldsPlugin`, `loglayer.LevelPlugin` convenience constructors for the common single-hook plugin cases.

### Go context integration

- `WithCtx(ctx)` attaches a `context.Context` to a single log call; surfaced to transports via `TransportParams.Ctx`.
- `loglayer.NewContext(ctx, log)` and `loglayer.FromContext(ctx)` embed and retrieve a logger from a Go context (zerolog-style middleware pattern). `MustFromContext` panics on absence.

### Thread safety

Every method on `*LogLayer` is safe to call from any goroutine, including concurrently with emission. `WithFields`, `WithoutFields`, `Child`, and `WithPrefix` return new loggers. Level toggling, transport changes, and mute toggles can all run live (SIGUSR1-driven debug toggling, hot-reloading transport lists) without any coordination on your side.

### Transports

Renderers (self-contained):

- `transports/console`: plain `fmt.Println`-style output to stdout/stderr.
- `transports/pretty`: colorized terminal output with five themes (Moonlight default, Sunlight, Neon, Nature, Pastel) and three view modes (inline, message-only, expanded). Pulls in `github.com/fatih/color`.
- `transports/structured`: one JSON object per log entry. Recommended for production.
- `transports/testing`: in-memory capture into a typed `LogLine` for test assertions.
- `transports/blank`: delegates `SendToLogger` to a user-supplied function. For prototyping or one-off integrations.

Network:

- `transports/http`: generic batched HTTP POST transport with a pluggable Encoder. Async worker drains a buffered channel into batches; configurable BatchSize, BatchInterval, BufferSize, Headers, Client, OnError. Foundation for service-specific wrappers like Datadog. Exposes `Close() error` to flush pending entries on shutdown.
- `transports/datadog`: Datadog Logs HTTP intake transport. Site-aware URL (us1/us3/us5/eu1/ap1), DD-API-KEY header, level → Datadog status mapping (debug/info/warning/error/critical), encoder producing the expected `{ddsource, service, hostname, ddtags, status, message, date, ...}` shape.

Logger wrappers:

- `transports/zerolog`: wraps `github.com/rs/zerolog`. Routes fatal entries through `WithLevel` so the core decides whether to exit.
- `transports/zap`: wraps `go.uber.org/zap`. Custom `CheckWriteHook` neutralizes zap's fatal-exit so the core decides via `DisableFatalExit`.
- `transports/slog`: wraps the stdlib `*log/slog.Logger`. Forwards `WithCtx` to `slog.Logger.LogAttrs`.
- `transports/phuslu`: wraps `github.com/phuslu/log`. **Always exits on Fatal** regardless of `DisableFatalExit`; phuslu calls `os.Exit` from any fatal dispatch path.
- `transports/logrus`: wraps `github.com/sirupsen/logrus`. Builds an internal copy with no-op `ExitFunc` so the user's logger is never mutated.
- `transports/charmlog`: wraps `github.com/charmbracelet/log`. Uses `Log(level, ...)` so the core controls the exit decision.
- `transports/otellog`: emits each entry as an OpenTelemetry `log.Record` on a `log.Logger` (`go.opentelemetry.io/otel/log`). Defaults to the global `LoggerProvider`; accepts an explicit `LoggerProvider` + `Name`/`Version`/`SchemaURL` or a pre-built `Logger`. Forwards `WithCtx` to `Logger.Emit` so SDK processors can correlate with the active span. Map metadata flattens to typed `KeyValue` attributes (recursing into `MapValue`/`SliceValue` for nested structures); struct metadata JSON-roundtrips into a nested `MapValue` under `MetadataFieldName` (default `"metadata"`).

### Integrations

- `integrations/loghttp`: HTTP middleware that derives a per-request logger from a base logger, attaches `requestId`/`method`/`path`, stores it in the request context via `loglayer.NewContext`, and emits a "request completed" log line with status, bytes, and duration. One line at server setup; downstream handlers retrieve the logger via `loghttp.FromRequest(r)`. Wraps any `net/http`-compatible router (chi, gorilla, gin, echo, stdlib). Functional options for request-ID header, request-ID generator, field names, status-based level escalation, optional start log, and an extra-fields hook.

### Plugins

- Plugin system with six lifecycle hooks: `OnFieldsCalled`, `OnMetadataCalled`, `OnBeforeDataOut`, `OnBeforeMessageOut`, `TransformLogLevel`, `ShouldSend` (per-transport gate). Plugins are function-field structs; populate the hook fields you want and `log.AddPlugin(loglayer.Plugin{...})`. Hook membership is pre-indexed at registration time so the dispatch path only walks plugins that actually implement the hook. Safe to add and remove from any goroutine, including concurrently with emission. Child loggers inherit plugins.
- Plugin hook panics are recovered centrally by the framework. A buggy plugin can't tear down the calling goroutine: each hook returns its no-op value on panic (nil for the `On*` shapes, level unchanged for `TransformLogLevel`, fail-open for `ShouldSend` so the entry still dispatches). Set `Plugin.OnError` to observe recovered panics; the framework wraps the recovered value in a hook-named error (`loglayer: plugin OnBeforeDataOut panicked: ...`) using `%w`, so callers can still `errors.Is`/`errors.As` against the original.
- All four dispatch-time hooks (`OnBeforeDataOut`, `OnBeforeMessageOut`, `TransformLogLevel`, `ShouldSend`) receive `Ctx context.Context` on their params, populated from `WithCtx`. Lets plugins read trace IDs, check cancellation, or otherwise make context-aware decisions per call.
- `(*LogLayer).WithCtx(ctx)` now returns `*LogLayer` and binds the context to every subsequent emission. The per-request HTTP handler pattern collapses from `log.WithCtx(r.Context()).Info(...)` on every line to `log = log.WithCtx(r.Context())` once. Builder-level `WithCtx` still overrides for a single emission. The `loghttp` middleware does the bind automatically — handlers reading via `loghttp.FromRequest(r)` get trace-aware logging with no boilerplate.
- `loglayer.MetadataPlugin`, `loglayer.FieldsPlugin`, `loglayer.LevelPlugin` convenience constructors for the common single-hook cases. Sugar over `loglayer.Plugin{ID: id, OnX: fn}`.
- `plugins/redact`: built-in redaction plugin. Match by `Keys` (exact key names; honors `json` tags when matching struct fields) or `Patterns` (regular expressions against string values). Walks nested maps, structs, slices, arrays, and pointers at any depth via reflection. Preserves the caller's runtime type: a struct in comes back as the same struct with sensitive fields replaced. Caller's input is never mutated. Dependency-free; works on both metadata and persistent fields.
- `plugins/datadogtrace`: Datadog APM trace injector plugin. Reads the active span from each entry's `WithCtx` context via a user-supplied `Extract` function and emits `dd.trace_id`, `dd.span_id`, plus optional `dd.service` / `dd.env` / `dd.version` for Datadog's log/trace correlation. Tracer-agnostic: works with `dd-trace-go` v1, v2, or any custom extractor; LogLayer itself takes no Datadog dependency.
- `plugins/oteltrace`: OpenTelemetry trace injector plugin. Reads the active span from each entry's `WithCtx` context via `trace.SpanContextFromContext` (`go.opentelemetry.io/otel/trace`) and emits `trace_id` / `span_id` in OTel's lowercase-hex form, plus optional `trace_flags`, `trace_state` (W3C vendor-specific routing/sampling info), and W3C baggage members under a configurable key prefix. Baggage rides independently of the trace span, so contexts with baggage but no active span still surface baggage attributes. Configurable keys (defaults match OTel's JSON serialization; switch to `trace.id`/`span.id` for ECS-style backends). Use with non-OTel transports for log/trace correlation; `transports/otellog` does this automatically when shipping through the OTel pipeline.

### Utilities

- `utils/maputil`: shared value-conversion and deep-clone primitives. `ToMap(any) map[string]any` normalizes any value to a flat map via JSON roundtrip. `Cloner.Clone(any) any` deep-clones a value with predicate-based key/value replacement at any depth, preserving the runtime type. Public so that third-party plugin and transport authors can build on the same foundation as the built-in packages.
