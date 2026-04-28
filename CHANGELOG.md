# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versioning
follows [SemVer](https://semver.org/spec/v2.0.0.html).

`go.loglayer.dev` is the main module. Three sub-modules ship under their
own tags: `go.loglayer.dev/transports/otellog`,
`go.loglayer.dev/plugins/oteltrace`, and (test-only)
`go.loglayer.dev/plugins/datadogtrace/livetest`. See `AGENTS.md` for the
splitting policy and release flow.

Releases are managed by [Release Please](https://github.com/googleapis/release-please)
from conventional commits. From v1.0.0 forward, this file is maintained
automatically; the `[Unreleased]` section below describes the initial
release at a high level.

## [Unreleased] (target: v1.0.0)

Initial release. Stable API; SemVer applies from this point forward.

LogLayer for Go is a transport-agnostic structured logging facade with a
fluent API for messages, fields, metadata, and errors. v1.0.0 ships:

- **Core**: `*LogLayer` with five log levels and a fluent builder
  (`WithMetadata`, `WithError`, `WithCtx`, `WithFields`, `WithGroup`,
  `WithPrefix`). Distinct `Fields`/`Metadata`/`Data` named types so the
  compiler catches misuse. `loglayer.F` and `loglayer.M` are short
  aliases for `Fields` and `Metadata` for terser call sites. Every
  method is safe to call from any goroutine, including concurrently
  with emission.
- **Renderers**: `pretty` (colorized terminal), `structured` (JSON per
  line), `console`, `testing`, `blank`.
- **Logger wrappers**: `zerolog`, `zap`, `log/slog`, `logrus`,
  `charmbracelet/log`, `phuslu/log`. Each forwards `WithCtx` so
  context-aware downstream handlers (OTel, etc.) keep working. Each
  vendor wrapper (zerolog/zap/logrus/phuslu/charmlog/otellog) ships
  as its own Go module so consumers only pay for the SDKs they
  actually import.
- **Network**: `http` (generic batched POST with pluggable Encoder),
  `datadog` (Logs HTTP intake with on-prem URL override; rejects
  non-https URLs by default, opt-in via `Config.AllowInsecureURL`).
- **OpenTelemetry**: `transports/otellog` (logs SDK) and
  `plugins/oteltrace` (trace_id/span_id injection on any transport).
  Both ship as separate Go modules so the OTel dep graph stays off
  users who don't import them.
- **Lifecycle**: `RemoveTransport` / `SetTransports` / `AddTransport`
  by-replace close the displaced transport if it implements
  `io.Closer` (HTTP/Datadog drain pending entries). Fatal-level
  emissions flush every transport before `os.Exit` so async logs
  aren't lost.
- **Plugins**: interface-based plugin system. `Plugin` is a one-method
  interface; six narrow hook interfaces (`FieldsHook`, `MetadataHook`,
  `DataHook`, `MessageHook`, `LevelHook`, `SendGate`) plus
  `ErrorReporter`. Adapter constructors (`NewFieldsHook`, etc.) for
  inline single-hook plugins; `WithErrorReporter` for adding panic
  observation to any plugin. Centralized panic recovery via
  `RecoveredPanicError`; default falls back to stderr when no
  `ErrorReporter` is implemented. Built-in plugins: `redact`,
  `datadogtrace`, `oteltrace`.
- **HTTP middleware**: `integrations/loghttp` derives a per-request
  logger, binds `r.Context()`, emits request-completed (or
  request-panicked) lines with status/duration/bytes.
- **slog interop**: `integrations/sloghandler` exposes a
  `log/slog.Handler` backed by a loglayer logger, so
  `slog.SetDefault(slog.New(sloghandler.New(log)))` makes every
  `slog.Info(...)` (yours and your dependencies') flow through
  loglayer's plugin pipeline, fan-out, groups, and level state.
  Complements `transports/slog` (which lets loglayer emit through a
  `*slog.Logger` backend); the new handler covers the opposite
  direction. Levels above `slog.LevelError` pin to `LogLevelError` so
  a slog emission cannot trigger Fatal exit.
- **Group routing**: name routing rules in `Config.Groups`, tag entries
  with `WithGroup(...)` to limit dispatch. Per-group level filters,
  active-groups env-var, runtime mutators.
- **Runtime control**: level mutators backed by atomic state for live
  toggling, transport hot-swap (atomic snapshot), plugin add/remove,
  mute fields/metadata.
- **Test helpers**: `transport/transporttest` exposes `RunContract` for
  the wrapper-transport contract suite; `plugins/plugintest` exposes
  `Install` / `AssertNoMutation` / `AssertPanicRecovered` for plugin
  authors. `transport/benchtest` exposes shared bench fixtures for
  per-module benchmarks.
- **fmtlog plugin**: optional `fmtlog.New()` plugin (in the `fmtlog`
  sub-package) that opts a logger into Sprintf semantics for
  multi-arg messages (`log.Info("user %d", id)`). Core stays
  structured-first; users opt in by registering the plugin.
- **Security defaults**: control-character sanitization on
  console/pretty messages and `loghttp` request headers; cycle-safe
  reflection in `maputil.Cloner`; `Datadog.Config` redacts the API key
  via `String()` and a `json:"-"` tag. The `redact` plugin now also
  walks the framework-built error subtree via `OnBeforeDataOut` so
  pattern-style redactors catch secrets baked into `err.Error()`. The
  `http` transport's default Client refuses cross-host redirects so
  credential headers (Authorization, X-API-Key, DD-API-KEY) cannot be
  forwarded to a redirected host. The HTTP worker recovers panics from
  user-supplied `Encoder` and `OnError` callbacks so a buggy callback
  cannot silently halt log delivery for the rest of the process.

Full API documented at <https://go.loglayer.dev>.

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main
