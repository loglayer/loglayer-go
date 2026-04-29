# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versioning
follows [SemVer](https://semver.org/spec/v2.0.0.html).

`go.loglayer.dev` is the main module — it hosts only the framework
core, the shared `transport/` package, and the `utils/*` helpers. Every
transport, plugin, and integration ships as its own sub-module under
its own tag (`<path>/v<X.Y.Z>`); the canonical list lives in
`.release-please-manifest.json`. See `AGENTS.md` for the layout and
release flow.

Releases are managed by [Release Please](https://github.com/googleapis/release-please)
from conventional commits. From v1.0.0 forward, this file is maintained
automatically; the `[Unreleased]` section below describes the initial
release at a high level.

## [1.2.0](https://github.com/loglayer/loglayer-go/compare/v1.1.1...v1.2.0) (2026-04-29)


### Features

* Cover Trace and Panic levels across transports and contract tests ([d511c7e](https://github.com/loglayer/loglayer-go/commit/d511c7eac484aadc3d3155876e497381b38f75e0))

## [1.1.1](https://github.com/loglayer/loglayer-go/compare/v1.1.0...v1.1.1) (2026-04-29)


### Code Refactoring

* Split the last four bundled packages so go.loglayer.dev hosts only the framework core ([#13](https://github.com/loglayer/loglayer-go/issues/13)) ([feef8a6](https://github.com/loglayer/loglayer-go/commit/feef8a6cc7b2350b802c43fc78b336037d4a5ea0))

## [1.1.0](https://github.com/loglayer/loglayer-go/compare/v1.0.2...v1.1.0) (2026-04-29)

> **Note on version**: this release is technically SemVer-breaking
> (the `go.loglayer.dev/fmtlog` import path was removed) but ships as
> v1.1.0 rather than v2.0.0. Reasoning: v1.0.x went out with no known
> users, and the post-split module layout (#11) means future breaking
> changes in any single transport/plugin/integration stay local to
> that sub-module's major version — the main `go.loglayer.dev` import
> path is intended to stay on v1 indefinitely. Callers using the old
> `fmtlog` import path should update per the migration entry below.
>
> All other loglayer-go modules (transports/*, plugins/*, integrations/*)
> were also tagged at v1.1.0 alongside this release as a coordinated
> bump. Their content is unchanged since their previous tags; the
> coordinated bump was a side-effect of the version-override commit
> applying globally to all packages release-please tracks. See each
> sub-module's `CHANGELOG.md` for the no-op v1.1.0 note.

### ⚠ BREAKING CHANGES

* **fmtlog:** the fmtlog plugin's import path changes from `go.loglayer.dev/fmtlog` to `go.loglayer.dev/plugins/fmtlog`. Update imports accordingly.

### Code Refactoring

* **fmtlog:** Move fmtlog from /fmtlog to /plugins/fmtlog ([#7](https://github.com/loglayer/loglayer-go/issues/7)) ([165cd4b](https://github.com/loglayer/loglayer-go/commit/165cd4b06cc8bbb290cf7d8d8930066a2c222f07))
* Split eight bundled sub-packages into independent modules ([#11](https://github.com/loglayer/loglayer-go/issues/11)) ([924d363](https://github.com/loglayer/loglayer-go/commit/924d3639ef44e89c726b635d6fec54efef9affb7))


### Miscellaneous Chores

* **release-please:** Expose refactor in changelog and force v2.0.0 ([#9](https://github.com/loglayer/loglayer-go/issues/9)) ([f3a980e](https://github.com/loglayer/loglayer-go/commit/f3a980e6eefa1e7b9f25a86d3f68de475de20534)), closes [#7](https://github.com/loglayer/loglayer-go/issues/7)
* **release-please:** Force next main release to v1.1.0 ([bbcd772](https://github.com/loglayer/loglayer-go/commit/bbcd77202f70ccd4f4a6f3bbe70f8dc91b4ea695))

## [1.0.2](https://github.com/loglayer/loglayer-go/compare/v1.0.1...v1.0.2) (2026-04-29)


### Bug Fixes

* **release-please:** Correct field name (tag-separator, not separator) ([37e1d1d](https://github.com/loglayer/loglayer-go/commit/37e1d1deba7e3c9e6197260e7df20f7b78f7a28e))

## [1.0.1](https://github.com/loglayer/loglayer-go/compare/v1.0.0...v1.0.1) (2026-04-29)


### Bug Fixes

* **charmlog:** Tidy go.sum ([c3d9832](https://github.com/loglayer/loglayer-go/commit/c3d9832920d84cf0d14a8d07febf251c595d60e3))
* **livetest:** Tidy go.sum ([5221684](https://github.com/loglayer/loglayer-go/commit/522168411257f5e89b1ae3c17ba086b94064b8ff))
* **otel:** Repair stale livetests ([de703c7](https://github.com/loglayer/loglayer-go/commit/de703c7f1e7131e018ccc5f80f5bfe1a451c1abc))
* **zap:** Tidy go.sum ([dc4d754](https://github.com/loglayer/loglayer-go/commit/dc4d7547a8f72cf81db8f7f05eb41e371160576a))

## 1.0.0 (2026-04-29)


### ⚠ BREAKING CHANGES

* **console:** emit logfmt key=value instead of Go map %v
* Config sub-structs, transport panic recovery, unified RecoveredPanicError
* **fmtlog:** rewrite as a MessageHook plugin
* split pretty/http/datadog into modules; per-module README/CHANGELOG
* split heavy-dep wrapper transports into sub-modules
* drop Trace level, swap to goccy/go-json, rewrite structured transport
* rework Plugin from struct-of-funcs to interface-based

### Features

* Caller / source-info capture (Config.AddSource); slog.Handler forwards Record.PC ([89106c4](https://github.com/loglayer/loglayer-go/commit/89106c43724321cd8aeedb2f945d4dc858b14c3d))
* Config sub-structs, transport panic recovery, unified RecoveredPanicError ([dc3d94f](https://github.com/loglayer/loglayer-go/commit/dc3d94f7bd7480cdc6ace661a368b1d0943a7835))
* **console:** Emit logfmt key=value instead of Go map %v ([8b29155](https://github.com/loglayer/loglayer-go/commit/8b291552372468c73c08d490be7beb415b0ab35a))
* Drop Trace level, swap to goccy/go-json, rewrite structured transport ([adb7114](https://github.com/loglayer/loglayer-go/commit/adb7114871931074b1e0c2b37de5466adf0e3ddc))
* Fmtlog sub-package for printf-style logging ([4e90b43](https://github.com/loglayer/loglayer-go/commit/4e90b435909d80a0429c5333180a5cfa2f03660e))
* Groups (named transport routing rules) ([a6ae557](https://github.com/loglayer/loglayer-go/commit/a6ae55718d3f0e7109918a215dab87cf307bed10))
* HTTP/Datadog transports, dev tooling, structure cleanup ([feaacc9](https://github.com/loglayer/loglayer-go/commit/feaacc9cae99581c836edafe70bd5738fdbc2e06))
* Initial v0.1.0 implementation ([bca353c](https://github.com/loglayer/loglayer-go/commit/bca353cf965645d46360cfb3b6f5e43bb4e135d0))
* Loglayer.F / loglayer.M aliases for Fields / Metadata ([624dbe4](https://github.com/loglayer/loglayer-go/commit/624dbe4b6307756fd70529789bd91bf7ecb85870))
* Optional auto-generated IDs for plugins and transports ([b6fc05a](https://github.com/loglayer/loglayer-go/commit/b6fc05a6c2d4a03359035c7f0aabbf9f77fca8d3))
* **otel:** Logs transport, trace plugin, livetests against real SDKs ([84d8e18](https://github.com/loglayer/loglayer-go/commit/84d8e189102a7d6c037894d5ccc0bd2405e5785a))
* **oteltrace:** Emit W3C trace_state and baggage members ([da40e25](https://github.com/loglayer/loglayer-go/commit/da40e25950942c8dbc2572ca57d5235cb83e5369))
* **plugins:** Datadogtrace plugin, persistent WithCtx, hook panic recovery ([3f701cf](https://github.com/loglayer/loglayer-go/commit/3f701cf69e081acb3caf6217b4c36707271d02bd))
* **plugins:** Plugin system, redact plugin, shared maputil ([b0cce53](https://github.com/loglayer/loglayer-go/commit/b0cce5376c56487064767f65692a69751aa76b6f))
* Rework Plugin from struct-of-funcs to interface-based ([0a95c1a](https://github.com/loglayer/loglayer-go/commit/0a95c1a3445b6cdfdc2669b3474ef092e154d318))
* Slog.Handler integration, security hardening, doc lead rewrite ([bcd4a1e](https://github.com/loglayer/loglayer-go/commit/bcd4a1e8679a28cb2454b78c1c04a198238d7a8f))
* Split heavy-dep wrapper transports into sub-modules ([aef6053](https://github.com/loglayer/loglayer-go/commit/aef6053a1e80c646eb2c666fd0f893b98a66e883))
* Split pretty/http/datadog into modules; per-module README/CHANGELOG ([852d363](https://github.com/loglayer/loglayer-go/commit/852d363829eb622032bf5608010f24755bba2c7c))
* Trace + Panic levels, stdlib log bridge, sampling plugins, error chain serializer ([78485ec](https://github.com/loglayer/loglayer-go/commit/78485ec55305d03a48f72ed89f8475471528659c))


### Bug Fixes

* 100go.co review — substring leaks, http transport TOCTOU/leaks ([ca4b08d](https://github.com/loglayer/loglayer-go/commit/ca4b08d8e2498f2921f5afc6fa9fa6adaa8b9924))
* Bound UnwrappingErrorSerializer walk against cyclic Unwrap ([57ca21b](https://github.com/loglayer/loglayer-go/commit/57ca21b2955627a443ce1cad5daeb7bdb13f48c2))
* Cap transport-close on Fatal/mutators; reject Datadog HTTP overrides ([feeb8fe](https://github.com/loglayer/loglayer-go/commit/feeb8fe9b3d46116eb65df31166763245249f81b))
* Datadog URL scheme; Close transports on remove/Fatal ([0f666ce](https://github.com/loglayer/loglayer-go/commit/0f666ce49ef8acb063c300284737c12d99c84645))
* **loghttp:** Recover handler panics + assorted doc clarifications ([14339e6](https://github.com/loglayer/loglayer-go/commit/14339e6106c73ae2a5a602057a2fbcacdae20c03))
* **loghttp:** Sanitize panic value before logging ([8bf612d](https://github.com/loglayer/loglayer-go/commit/8bf612ddbb66f6072d88bf0793f093a146dab74b))
* MetadataOnly plugin-set straddle; trim narrating comments ([0de80d8](https://github.com/loglayer/loglayer-go/commit/0de80d837b956b2660818695f06909d13f17ed36))
* Races, walker correctness, and builder plugin-set straddle ([d150c3c](https://github.com/loglayer/loglayer-go/commit/d150c3c13fe24fe37c1a0f5b9b700c4f680f370c))
* **security:** Cycle-safe Cloner, header sanitization, secret redaction + on-prem Datadog URL ([057c787](https://github.com/loglayer/loglayer-go/commit/057c787681843e611fdf04a83db64cdeed429d7e))


### Performance Improvements

* Byte-scan fast path for SanitizeMessage + benchmark coverage ([9dbbec7](https://github.com/loglayer/loglayer-go/commit/9dbbec7b7fec48914ff8b8e7336dffb7820603aa))
* **maputil:** Reflect-based struct walker for ToMap ([3881861](https://github.com/loglayer/loglayer-go/commit/38818614454c0d65a4494f722828ec2e51f2cafe))


### Code Refactoring

* **fmtlog:** Rewrite as a MessageHook plugin ([655bcf6](https://github.com/loglayer/loglayer-go/commit/655bcf65d5a6572f556829326556e59dcd570978))

## [Unreleased] (target: v1.0.0)

Initial release. Stable API; SemVer applies from this point forward.

LogLayer for Go is a transport-agnostic structured logging facade with a
fluent API for messages, fields, metadata, and errors. v1.0.0 ships:

- **Core**: `*LogLayer` with seven log levels (Trace, Debug, Info, Warn,
  Error, Fatal, Panic) and a fluent builder (`WithMetadata`, `WithError`,
  `WithContext`, `WithFields`, `WithGroup`, `WithPrefix`). Trace sits below
  Debug for fine-grained diagnostic; Panic dispatches and then panics
  with the joined message string (recoverable, matching zerolog/zap/
  logrus convention). Distinct `Fields`/`Metadata`/`Data` named types
  so the compiler catches misuse. `loglayer.F` and `loglayer.M` are
  short aliases for `Fields` and `Metadata` for terser call sites.
  Every method is safe to call from any goroutine, including
  concurrently with emission. The level set is fixed at this release;
  the per-level bitmap is a `uint32` with headroom for ~24 more, and
  the three extension points (`levelIndex`, `LogLevel.String`,
  `ParseLogLevel`) are switches that can be replaced with a registry
  lookup later if user-registered custom levels become a need.
- **Renderers**: `pretty` (colorized terminal), `structured` (JSON per
  line), `console`, `testing`, `blank`.
- **Logger wrappers**: `zerolog`, `zap`, `log/slog`, `logrus`,
  `charmbracelet/log`, `phuslu/log`. Each forwards `WithContext` so
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
- **Transport panic recovery (opt-in)**: `Config.OnTransportPanic`,
  when set, makes the dispatch loop recover panics from a
  transport's `SendToLogger`, report them via the callback, and
  continue dispatch to the remaining transports. Default (nil) is
  unchanged from prior behavior: a panicking transport propagates
  up, matching Go logging convention. Off-by-default keeps the hot
  path a direct call (the recover wrap costs ~8 ns per emission per
  transport when on). The callback receives a `*RecoveredPanicError`
  matching the shape plugin hooks already surface via
  `ErrorReporter.OnError`, so a single observability function can
  absorb panics from either source. `Kind` (PanicKindPlugin or
  PanicKindTransport) distinguishes the origin; `Hook` is the
  specific identifier (hook name or transport ID).
- **Plugins**: interface-based plugin system. `Plugin` is a one-method
  interface; six narrow hook interfaces (`FieldsHook`, `MetadataHook`,
  `DataHook`, `MessageHook`, `LevelHook`, `SendGate`) plus
  `ErrorReporter`. Adapter constructors (`NewFieldsHook`, etc.) for
  inline single-hook plugins; `WithErrorReporter` for adding panic
  observation to any plugin. Centralized panic recovery via
  `RecoveredPanicError`; default falls back to stderr when no
  `ErrorReporter` is implemented. Built-in plugins: `redact`,
  `sampling` (`FixedRate`, `FixedRatePerLevel`, `Burst`),
  `datadogtrace`, `oteltrace`.
- **HTTP middleware**: `integrations/loghttp` derives a per-request
  logger, binds `r.Context()`, emits request-completed (or
  request-panicked) lines with status/duration/bytes.
- **stdlib log bridge**: `*LogLayer.Writer(level)` returns an `io.Writer`
  and `*LogLayer.NewLogLogger(level)` returns a `*log.Logger`, both
  emitting one entry per Write through the full pipeline. Drop into
  `http.Server.ErrorLog`, gorm, database/sql tracing, or anything that
  takes a `*log.Logger` or an `io.Writer`. Mirrors `slog.NewLogLogger`.
- **Error chain expansion**: opt-in `loglayer.UnwrappingErrorSerializer`
  walks `errors.Unwrap` and `errors.Join`'s `Unwrap() []error` to surface
  every wrapped cause as a `causes` array on the serialized error. The
  default serializer is unchanged.
- **slog interop**: `integrations/sloghandler` exposes a
  `log/slog.Handler` backed by a loglayer logger, so
  `slog.SetDefault(slog.New(sloghandler.New(log)))` makes every
  `slog.Info(...)` (yours and your dependencies') flow through
  loglayer's plugin pipeline, fan-out, groups, and level state.
  Complements `transports/slog` (which lets loglayer emit through a
  `*slog.Logger` backend); the new handler covers the opposite
  direction. Levels above `slog.LevelError` pin to `LogLevelError` so
  a slog emission cannot trigger Fatal exit. Source info is forwarded
  automatically: `slog.Record.PC` becomes a `*loglayer.Source` via
  `RawLogEntry.Source`, no `Config.Source.Enabled` needed.
- **Source / caller info**: opt-in capture of file/line/function for
  every emission via `Config.Source.Enabled` (paired with
  `Config.Source.FieldName`, default `"source"`). Surfaced in the
  assembled `Data`, with JSON tags matching the slog convention so structured
  output is interchangeable. Cost is one `runtime.Caller` per
  emission, paid only when on. The `Source` struct also implements
  `fmt.Stringer` (compact `func file:line`) and `slog.LogValuer`
  (nested group) so non-JSON transports render readably. Adapters
  with their own PC can pass it via `RawLogEntry.Source` to skip the
  runtime walk; a `loglayer.SourceFromPC` helper builds a Source
  from a captured PC.
- **Group routing**: name routing rules in `Config.Routing.Groups`, tag entries
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
