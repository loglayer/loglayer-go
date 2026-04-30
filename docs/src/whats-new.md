---
title: What's new in LogLayer for Go
description: Latest features and improvements in LogLayer for Go.
---

# What's new in LogLayer for Go

- See the [main `CHANGELOG.md`](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md) for the auto-generated per-release log.

## Apr 29, 2026

`loglayer`:

- **Lazy evaluation**: `loglayer.Lazy(fn)` defers value computation in `WithFields` until log dispatch time. The callback runs only when the level passes and re-evaluates on every emission, so expensive work (memory stats, request summaries, large struct serialization) costs nothing on filtered-out entries. Adapted from [LogTape's lazy evaluation](https://logtape.org/manual/lazy). See [Lazy Evaluation](/logging-api/lazy-evaluation).
- **Groups exposed to transports and plugin hooks**: `TransportParams.Groups` and the four dispatch-time hook param structs (`BeforeDataOutParams`, `BeforeMessageOutParams`, `TransformLogLevelParams`, `ShouldSendParams`) now carry the merged set of persistent and per-call `WithGroup` tags for the entry. Routing decisions still happen before any hook fires; the slice is exposed so transports can ship groups in the wire payload and plugins can drive group-aware data, message, level, or send-gate transformations. Matches the TypeScript loglayer's surface (`LogLayerTransportParams.groups`, `PluginShouldSendToLoggerParams.groups`) and goes beyond it for the other dispatch-time hooks. See [Reading params.Groups](/transports/creating-transports#reading-params-groups) and [Per-call groups](/plugins/creating-plugins#per-call-groups).

`transports/lumberjack`:

- **Initial release**: new file transport that writes one JSON object per log entry to a rotating file via [lumberjack.v2](https://github.com/natefinch/lumberjack). Size-triggered rollover, configurable backup retention, age-based cleanup, optional gzip compression, and a `Rotate()` method for SIGHUP-driven roll-overs. The `lumberjack` suffix names the rotation backend explicitly so the shorter `transports/file` name stays available for a future rolled-our-own implementation. See [File (Lumberjack)](/transports/lumberjack).

`v1.1.0`:

- **Multi-module split**: every transport (`transports/*`), plugin (`plugins/*`), and integration (`integrations/*`) now ships as its own independently-versioned Go module. Import paths are unchanged; you may need a `go mod tidy` to pick up the new sub-module entries. Future breaking changes in any one sub-package bump only that sub-module's major version, leaving the main `go.loglayer.dev` import path stable on v1. Full module list in [`.release-please-manifest.json`](https://github.com/loglayer/loglayer-go/blob/main/.release-please-manifest.json).
- **`fmtlog` import path moved**: the `fmtlog` plugin moved from `go.loglayer.dev/fmtlog` to `go.loglayer.dev/plugins/fmtlog` for consistency with every other plugin. Update imports:

  ```diff
  - import "go.loglayer.dev/fmtlog"
  + import "go.loglayer.dev/plugins/fmtlog"
  ```

  Technically a SemVer-breaking change (the old import path is gone), but ships as a minor bump rather than v2.0.0. The v1.0.x release went out with no known users yet.

`v1.0.0`:

- **Initial release**: LogLayer for Go is a transport-agnostic structured logging facade: one fluent API on top of any logging library, JSON or pretty rendering, HTTP shipping, OpenTelemetry, or your own transport. Stable API; SemVer applies from this point forward. See [Getting Started](/getting-started).
