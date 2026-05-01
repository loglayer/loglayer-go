---
title: What's new in LogLayer for Go
description: Latest features and improvements in LogLayer for Go.
---

# What's new in LogLayer for Go

- See the [main `CHANGELOG.md`](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md) for the auto-generated per-release log.

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
