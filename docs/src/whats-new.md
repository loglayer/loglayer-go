---
title: What's new in LogLayer for Go
description: Latest features and improvements in LogLayer for Go.
---

# What's new in LogLayer for Go

- See the [main `CHANGELOG.md`](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md) for the auto-generated per-release log.

## May 02, 2026

`v2.0.0`:

**Breaking: import paths bump to `/v2`.** The loglayer core no longer mutates `Messages[0]` to fold the `WithPrefix` value into the message text. The prefix flows through `TransportParams.Prefix` and each transport decides how to render it. Built-in transports preserve v1 user-visible output via the new `transport.JoinPrefixAndMessages` helper; the cli transport opts into smart rendering (dim-grey user prefix separate from level color). See [Migrating to v2](/migrating-to-v2) for the upgrade checklist.

`loglayer`:

`Prefix` is now exposed as a separate field on `TransportParams` and on every dispatch-time plugin hook param struct (`BeforeDataOutParams`, `BeforeMessageOutParams`, `TransformLogLevelParams`, `ShouldSendParams`). Transports and plugins can render or react to the prefix independently from the message string. The legacy "prepend prefix into `Messages[0]`" auto-mutation in v1.7.x stays in place for backwards compatibility within the v1 line; v2.0.0 removes it.

`transports/cli`:

Initial release. New [CLI transport](/transports/cli) tuned for command-line app output: short level prefixes, stdout / stderr routing, TTY-detected ANSI color, no timestamps. Includes table rendering for slice-of-map metadata so the same call site emits a CLI table and a JSON array depending on the transport.

New `Config.TableColumnOrder []string` knob pins the leading column order for slice-of-map metadata table rendering. Keys named here render in the listed order; the rest sort lexicographically and follow. Empty / nil keeps the previous fully-lexicographic behavior. See [Pinning column order](/transports/cli#pinning-column-order).

`transports/http`:

New `Config.String()` redacts `Headers` values so an accidental `log.Info(cfg)` or `fmt.Sprintf("%v", cfg)` can't leak credentials passed via `Authorization` / `X-API-Key` / similar headers. Header keys stay visible for debuggability. Mirrors the redaction shape already used by `transports/datadog`.

`defaultCheckRedirect` now compares hosts case-insensitively, so legitimate same-host redirects with mixed-case URLs aren't refused. Cross-host refusal still applies; ports are still compared exactly.

New `Config.ShutdownTimeout` (default 5s) bounds how long `Close` waits for in-flight requests to finish during shutdown. When the timeout elapses, the worker's outbound HTTP requests are cancelled via context so `Close` can return even if the endpoint is wedged; previously a stuck endpoint could pin `Close` for up to the per-request `Client.Timeout` (30s default), and the parent `flushTransports`'s 5s timeout would leak the close goroutine.

## Apr 30, 2026

`transports/gcplogging`:

Initial release. New [Google Cloud Logging transport](/transports/gcplogging).

`transports/sentry`:

Initial release. New [Sentry transport](/transports/sentry).

`loglayer`:

**`MetadataFieldName` is now a core `Config` knob.** Set it once on `loglayer.Config` and metadata nests under the configured key uniformly across every transport, for both map and non-map values. Joins `FieldsKey` and `ErrorFieldName` as the third assembly-shape knob. See [`MetadataFieldName`](/configuration#metadatafieldname).

The resolved assembly shape (`FieldsKey`, `MetadataFieldName`, `ErrorFieldName`, `SourceFieldName`) is also published as `loglayer.Schema` on `TransportParams` and on the dispatch-time plugin hook param structs (`BeforeDataOutParams`, `BeforeMessageOutParams`, `TransformLogLevelParams`, `ShouldSendParams`). Plugins can navigate `params.Data` precisely without guessing keys.

The per-transport `Config.MetadataFieldName` field is removed from every wrapper (zerolog, zap, charmlog, phuslu, logrus, slog, otellog, sentry); set the core knob instead.

`transports/pretty`:

- **Column-aligned YAML in expanded mode**: same-level scalar keys pad to the longest sibling so values line up. Multi-line keys (nested maps, slices) don't participate in the alignment width.
- **Nested rendering for keyed metadata**: when `Config.MetadataFieldName` is set, the metadata value JSON-roundtrips into a nested map and renders as YAML under the configured key.

## Apr 29, 2026

`loglayer`:

- **Lazy evaluation**: `loglayer.Lazy(fn)` defers value computation in `WithFields` until log dispatch time. The callback runs only when the level passes and re-evaluates on every emission, so expensive work costs nothing on filtered-out entries. See [Lazy Evaluation](/logging-api/lazy-evaluation).
- **Groups in dispatch-time hooks**: `TransportParams.Groups` and the four dispatch-time plugin hook param structs now carry the entry's merged `WithGroup` tags. Routing still happens before hooks fire; the slice is exposed so transports can ship groups in the wire payload and plugins can drive group-aware transformations. See [Reading params.Groups](/transports/creating-transports#reading-params-groups).

`transports/lumberjack`:

Initial release. New [rotating-file transport](/transports/lumberjack).

`v1.1.0`:

**Multi-module split.** Every transport (`transports/*`), plugin (`plugins/*`), and integration (`integrations/*`) now ships as its own independently-versioned Go module. Import paths are unchanged; you may need `go mod tidy` to pick up the new sub-module entries. Future breaking changes in any sub-package bump only that module's tag namespace, leaving `go.loglayer.dev` itself stable on v1. Full module list in [`monorel.toml`](https://github.com/loglayer/loglayer-go/blob/main/monorel.toml).

**`fmtlog` import path moved.** The plugin moved from `go.loglayer.dev/fmtlog` to `go.loglayer.dev/plugins/fmtlog` for consistency with every other plugin. Update imports:

```diff
- import "go.loglayer.dev/fmtlog"
+ import "go.loglayer.dev/plugins/fmtlog"
```

Technically a SemVer-breaking change, but ships as a minor bump rather than v2.0.0 (the v1.0.x release went out with no known users yet).

`v1.0.0`:

Initial release. LogLayer for Go is a transport-agnostic structured logging facade: one fluent API on top of any logging library, JSON or pretty rendering, HTTP shipping, OpenTelemetry, or your own transport. Stable API; SemVer applies from this point forward. See [Getting Started](/getting-started).
