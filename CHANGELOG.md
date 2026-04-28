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
  compiler catches misuse. Every method is safe to call from any
  goroutine, including concurrently with emission.
- **Renderers**: `pretty` (colorized terminal), `structured` (JSON per
  line), `console`, `testing`, `blank`.
- **Logger wrappers**: `zerolog`, `zap`, `log/slog`, `logrus`,
  `charmbracelet/log`, `phuslu/log`. Each forwards `WithCtx` so
  context-aware downstream handlers (OTel, etc.) keep working.
- **Network**: `http` (generic batched POST with pluggable Encoder),
  `datadog` (Logs HTTP intake with on-prem URL override).
- **OpenTelemetry**: `transports/otellog` (logs SDK) and
  `plugins/oteltrace` (trace_id/span_id injection on any transport).
  Both ship as separate Go modules so the OTel dep graph stays off
  users who don't import them.
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
- **Group routing**: name routing rules in `Config.Groups`, tag entries
  with `WithGroup(...)` to limit dispatch. Per-group level filters,
  active-groups env-var, runtime mutators.
- **Runtime control**: level mutators backed by atomic state for live
  toggling, transport hot-swap (atomic snapshot), plugin add/remove,
  mute fields/metadata.
- **Test helpers**: `transport/transporttest` exposes `RunContract` for
  the wrapper-transport contract suite; `plugins/plugintest` exposes
  `Install` / `AssertNoMutation` / `AssertPanicRecovered` for plugin
  authors.
- **Security defaults**: control-character sanitization on
  console/pretty messages and `loghttp` request headers; cycle-safe
  reflection in `maputil.Cloner`; `Datadog.Config` redacts the API key
  via `String()` and a `json:"-"` tag.

Full API documented at <https://go.loglayer.dev>.

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main
