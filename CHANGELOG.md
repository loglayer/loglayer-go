# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The repo is currently a single Go module: `go.loglayer.dev`. All packages
move together under one tag. If a transport later needs an independent release
cadence we may split it into its own module; see `AGENTS.md` for the policy.

## [Unreleased] (target: v0.1.0)

Initial release. Pre-1.0; the API may still shift before v1.

### Core

- `loglayer.LogLayer` with the six standard log levels (Trace, Debug, Info,
  Warn, Error, Fatal) and a fluent builder API: `WithMetadata`, `WithError`,
  `WithCtx`, chained into a level method (`Info`, `Warn`, etc.).
- Persistent fields via `WithFields(loglayer.Fields)` and `WithoutFields(keys...)`.
  Both return a new logger; the receiver is unchanged. Matches the convention
  of zerolog, zap, slog, and logrus.
- `loglayer.Fields` and `loglayer.Metadata` type aliases for `map[string]any`.
  `WithMetadata` accepts `any`; transports decide how to render structs and
  other types.
- `Child()` and `WithPrefix(prefix)` for explicit cloning.
- `Raw(RawLogEntry)` for forwarding pre-assembled entries from another system.
- `MetadataOnly` and `ErrorOnly` shortcuts.
- `loglayer.New(Config)` panics on misconfiguration; `loglayer.Build(Config)`
  returns `loglayer.ErrNoTransport` instead.
- `Config.DisableFatalExit` opts out of `os.Exit(1)` after a Fatal log
  (default-on, matching Go convention).
- `loglayer.NewMock()` returns a silent `*LogLayer` for tests with
  `DisableFatalExit` enabled automatically.
- Go context integration: `WithCtx(ctx)` per-call, surfaced via
  `TransportParams.Ctx`. `loglayer.NewContext(ctx, log)` and
  `loglayer.FromContext(ctx)` for embedding/retrieving a logger inside a Go
  context (zerolog-style middleware pattern). `MustFromContext` panics on
  absence.

### Configuration

- `Config.Disabled bool` replaces the prior `Config.Enabled *bool`. Same for
  `transport.BaseConfig.Disabled bool` (was `Enabled *bool`). The `*bool`
  trick was un-idiomatic; the negated field works because the zero value
  (`false`) means "logger is on", matching the runtime
  `EnableLogging`/`DisableLogging` methods.
- `Config{Transport: ..., Transports: ...}` is now rejected at construction
  with `ErrTransportAndTransports`. Previously `Transport` silently won.
- `MetadataOnly` now takes `MetadataOnlyOpts` (was a variadic `LogLevel`),
  matching the existing `ErrorOnly` shape. Old:
  `log.MetadataOnly(m, loglayer.LogLevelWarn)`. New:
  `log.MetadataOnly(m, loglayer.MetadataOnlyOpts{LogLevel: loglayer.LogLevelWarn})`.
- `transports/console.ConsoleTransport` and
  `transports/structured.StructuredTransport` renamed to `Transport` to
  match the rest of the transports (no more `console.ConsoleTransport`
  stutter).
- New `ErrPluginNoID` sentinel. `Build` now returns it instead of
  panicking when a `Config.Plugins` entry has an empty ID. `New` and
  `AddPlugin` still panic, but with the sentinel.
- `transports/http` and `transports/datadog` gain `Build(Config) (*Transport, error)`
  siblings to their panic-on-misconfig `New`. New sentinels:
  `httptransport.ErrURLRequired` and `datadog.ErrAPIKeyRequired`. Use
  `Build` when loading required values from environment variables.
- `TransportParams.HasData` removed (was always `Data != nil`). Use
  `len(params.Data) > 0` directly. Same for `transports/testing.LogLine`.
- `integrations/loghttp.Middleware` now takes `loghttp.Config` instead
  of variadic functional options, matching every other package in the
  library. Old: `loghttp.Middleware(log, loghttp.WithStartLog(true))`.
  New: `loghttp.Middleware(log, loghttp.Config{StartLog: true})`. The
  `WithXxx` Option constructors are removed.
- `transports/zerolog.Config.DisableFatalExit` removed. The field was
  declared but never read; the rationale ("the transport always honors
  loglayer's contract") moved to the package doc comment.
- `ErrorOnlyOpts.CopyMsg` is now a typed `CopyMsgPolicy` enum
  (`CopyMsgDefault`, `CopyMsgEnabled`, `CopyMsgDisabled`) instead of
  `*bool`. The zero value defers to `Config.CopyMsgOnOnlyError`. Old:
  `b := true; opts.CopyMsg = &b`. New: `opts.CopyMsg = loglayer.CopyMsgEnabled`.
- `WithFreshTransports` renamed to `SetTransports`. The method mutates
  the receiver in place; the prior `With*` prefix violated the
  documented "With* returns a new logger" convention.
- `AddPlugin` is now variadic, matching `AddTransport`. Old:
  `log.AddPlugin(p)`. New: same call works, plus `log.AddPlugin(p1, p2, p3)`
  and `log.AddPlugin(plugins...)`.
- `transports/http` worker errors now include the package prefix
  (`loglayer/transports/http: encode: ...` rather than just `encode: ...`)
  so callers receiving them via `OnError` can identify the source.
- Removed obsolete `PLAN.md` from the repo root.
- `ClearFields` renamed to `WithoutFields`. The method returns a new
  logger; the prior `Clear*` prefix violated the documented "With*
  returns a new logger" convention. Matches the TypeScript loglayer's
  `withoutContext` precedent. Migration: `log = log.ClearFields(...)`
  becomes `log = log.WithoutFields(...)`.

### Groups

- Group routing: named rules that decide which transports receive
  which log entries based on tags. Mirrors the TypeScript loglayer's
  `withGroup` feature.
- `Config.Groups map[string]LogGroup` defines named groups. Each group
  lists the transport IDs it routes to plus an optional minimum level
  and a `Disabled` toggle.
- `Config.ActiveGroups []string` restricts routing to a named subset;
  nil/empty means "no filter."
- `Config.UngroupedRouting` controls entries with no group tag. Three
  modes: `UngroupedToAll` (default, backwards-compatible),
  `UngroupedToNone`, `UngroupedToTransports` (allowlist).
- Per-call tagging via `(*LogBuilder).WithGroup(groups ...string)`.
- Persistent tagging via `(*LogLayer).WithGroup(groups ...string)`,
  which returns a child logger. Tags accumulate across chained calls
  (deduplicated).
- Runtime mutators: `AddGroup`, `RemoveGroup`, `EnableGroup`,
  `DisableGroup`, `SetGroupLevel`, `SetActiveGroups`,
  `ClearActiveGroups`, `GetGroups`. All safe from any goroutine
  (atomic publish + mutex-serialized writers).
- `Raw(RawLogEntry{Groups: ...})` overrides assigned groups for
  forwarded entries.
- `loglayer.ActiveGroupsFromEnv(envName)` parses a comma-separated
  group list from an environment variable. Use it explicitly to set
  `Config.ActiveGroups` from `LOGLAYER_GROUPS` or similar; the library
  does not read environment variables on your behalf.

### Thread safety

Every method on `*LogLayer` is safe to call from any goroutine, including
concurrently with emission. There is no setup-only category. See
`AGENTS.md#thread-safety` for the per-method breakdown:

- Returns-new methods (`WithFields`, `WithoutFields`, `Child`, `WithPrefix`)
  build a new logger; receiver untouched.
- Level mutators backed by `atomic.Uint32` bitmap (mirrors `zap.AtomicLevel`).
- Transport mutators backed by `atomic.Pointer[transportSet]`; concurrent
  mutators on the same logger serialize via an internal mutex.
- Mute toggles backed by `atomic.Bool` state.

### Transports

Renderers (self-contained):

- `transports/console`: plain `fmt.Println`-style output to stdout/stderr.
- `transports/pretty`: colorized terminal output with five themes
  (Moonlight, Sunlight, Neon, Nature, Pastel) and three view modes
  (inline, message-only, expanded). Pulls in `github.com/fatih/color`.
- `transports/structured`: one JSON object per log entry.
- `transports/testing`: in-memory capture for test assertions.
- `transports/blank`: delegates `SendToLogger` to a user-supplied function.
  For prototyping or one-off integrations.

Network:

- `transports/http`: generic batched HTTP POST transport. Async worker drains
  a buffered channel into batches; configurable BatchSize, BatchInterval,
  BufferSize, Headers, Client, Encoder, OnError. Default JSONArrayEncoder
  produces `[{level, time, msg, ...fields, metadata?}, ...]`. Exposes
  `Close() error` to flush on shutdown.
- `transports/datadog`: Datadog Logs HTTP intake wrapper around
  `transports/http`. Site-aware URL (us1/us3/us5/eu1/ap1), DD-API-KEY header,
  encoder producing Datadog's expected shape (ddsource, service, hostname,
  ddtags, status, message, date) with level → status mapping.

Logger wrappers:

- `transports/zerolog`: wraps `github.com/rs/zerolog`. Routes fatal entries
  through `WithLevel` so the core decides whether to exit.
- `transports/zap`: wraps `go.uber.org/zap`. Custom `CheckWriteHook`
  neutralizes zap's fatal-exit so the core decides via `DisableFatalExit`.
- `transports/slog`: wraps the stdlib `*log/slog.Logger`. Forwards `WithCtx`
  to `slog.Logger.LogAttrs`.
- `transports/phuslu`: wraps `github.com/phuslu/log`. Note: phuslu cannot be
  prevented from calling `os.Exit` on Fatal; documented as a limitation.
- `transports/logrus`: wraps `github.com/sirupsen/logrus`. Builds an internal
  copy with no-op `ExitFunc` so the user's logger is never mutated.
- `transports/charmlog`: wraps `github.com/charmbracelet/log`. Uses
  `Log(level, ...)` to defer the exit decision to the core.
- `transports/otellog`: emits each entry as an OpenTelemetry
  `log.Record` on a `log.Logger` (`go.opentelemetry.io/otel/log`).
  Defaults to `global.GetLoggerProvider`; accepts an explicit
  `LoggerProvider` (with `Name`, `Version`, `SchemaURL`) or a pre-built
  `Logger` for tests/advanced wiring. Forwards `WithCtx` to `Logger.Emit`
  so SDK processors can correlate the record with the active span. Map
  metadata flattens to typed `KeyValue` attributes (with recursive
  `MapValue`/`SliceValue` for nested structures); struct metadata
  JSON-roundtrips into a nested `MapValue` under `MetadataFieldName`
  (default `"metadata"`).

### Integrations

- `integrations/loghttp`: HTTP middleware that derives a per-request logger
  with `requestId`/`method`/`path` fields, stores it in the request context,
  and emits a "request completed" log with status, bytes, and duration.
  Functional options for request-ID header, request-ID generator, field
  names, status-based level escalation, optional start log, extra-fields
  hook. Wraps any `net/http`-compatible router (chi, gorilla, gin, echo,
  stdlib).

### Plugins

- Plugin system with six lifecycle hooks: `OnFieldsCalled`,
  `OnMetadataCalled`, `OnBeforeDataOut`, `OnBeforeMessageOut`,
  `TransformLogLevel`, `ShouldSend` (per-transport gate). Plugins are
  function-field structs registered via `*LogLayer.AddPlugin`. Hook
  membership is pre-indexed at registration time; dispatch path pays only
  for hooks that are populated. Safe to add/remove from any goroutine.
  Child loggers inherit plugins.
- Plugin hook panics are recovered centrally by the framework so a
  buggy plugin can't tear down the caller's goroutine. Each hook
  returns its no-op value on panic (nil / level unchanged / fail-open
  for `ShouldSend`). Set `Plugin.OnError` to observe recovered panics;
  the framework wraps the recovered value in a hook-named error
  (`loglayer: plugin <hook> panicked: ...`) using `%w` so callers can
  still `errors.Is`/`errors.As` against the original.
- All four dispatch-time hooks
  (`OnBeforeDataOut`, `OnBeforeMessageOut`, `TransformLogLevel`,
  `ShouldSend`) receive `Ctx context.Context` on their params, populated
  from `WithCtx`. Lets plugins read trace IDs, check cancellation, etc.
- `(*LogLayer).WithCtx(ctx)` now returns `*LogLayer` (was `*LogBuilder`)
  and binds the context to every subsequent emission. Mirrors the
  `WithGroup` pattern: same name, persistent on the logger,
  per-call-overridable on a builder. The dominant per-request handler
  pattern collapses from "call WithCtx on every emission" to "bind once."
  Builder chains like `log.WithCtx(ctx).WithMetadata(...).Info(...)`
  still work transparently.
- `loghttp.Middleware` automatically binds `r.Context()` to the
  per-request logger. Handlers reading via `loghttp.FromRequest(r)` get
  trace-aware logging without any per-emission boilerplate.
- Convenience constructors `loglayer.MetadataPlugin`,
  `loglayer.FieldsPlugin`, `loglayer.LevelPlugin` for the common
  single-hook cases.
- `plugins/redact`: built-in redaction plugin. Matches by `Keys`
  (exact key names, json-tag aware for struct fields) or `Patterns`
  (regular expressions against string values). Walks nested maps,
  structs, slices, and pointers at any depth via reflection;
  preserves the caller's runtime type (struct in → struct out).
  Caller's input is never mutated. Dependency-free.
- `plugins/datadogtrace`: Datadog APM trace injector plugin. Reads
  the active span from each entry's `WithCtx` context via a
  user-supplied `Extract` function and emits `dd.trace_id`,
  `dd.span_id`, plus optional `dd.service` / `dd.env` / `dd.version`
  for Datadog's log/trace correlation. Tracer-agnostic: works with
  dd-trace-go v1, v2, or any custom extractor; LogLayer itself takes
  no Datadog dependency.
- `plugins/oteltrace`: OpenTelemetry trace injector plugin. Reads the
  active span from each entry's `WithCtx` context via
  `trace.SpanContextFromContext` and emits `trace_id` / `span_id`
  (configurable keys) plus optional `trace_flags`, `trace_state` (W3C
  vendor-specific routing/sampling info), and W3C baggage members
  under a configurable key prefix (`baggage.user_id` etc.). Baggage
  rides independently of the trace span, so contexts with baggage but
  no active span still surface baggage attributes. Use with non-OTel
  transports for log/trace correlation; the OTel pipeline does this
  itself when shipping via `transports/otellog`.

### Utilities

- `utils/maputil`: shared value-conversion and deep-clone helpers.
  `ToMap(any)` normalizes any value to `map[string]any` via JSON
  roundtrip. `Cloner{MatchKey, MatchValue, Censor}.Clone(any)` deep-
  clones a value with replacement predicates applied at any depth,
  preserving the runtime type. Used by the structured/pretty/datadog
  transports and the redact plugin; available to third-party plugins
  and transports.

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main
