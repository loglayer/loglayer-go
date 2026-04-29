---
title: What's new in LogLayer for Go
description: Latest features and improvements in LogLayer for Go.
---

# What's new in LogLayer for Go

- [`go.loglayer.dev` Changelog](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md)
- For users coming from the TypeScript [`loglayer`](https://loglayer.dev) library, see [For TypeScript Developers](/for-typescript-developers) for the API mapping.

## File transport (rotating)

`transports/lumberjack` ships as its own module: one JSON object per log entry, written to a rotating file via [lumberjack.v2](https://github.com/natefinch/lumberjack). Size-triggered rollover, configurable backup retention, age-based cleanup, optional gzip compression, and a `Rotate()` method for SIGHUP-driven roll-overs. See [File (Lumberjack)](/transports/lumberjack).

The `lumberjack` suffix names the rotation backend explicitly so the shorter `transports/file` name stays available for a future rolled-our-own implementation. If you want pretty/console output written to a rotating file today, you don't need this transport. Pass a `*lumberjack.Logger` directly as the `Writer` field of any existing transport.

## Apr 29, 2026

`v1.0.0`:

- Initial release. LogLayer for Go is a transport-agnostic structured logging facade — one fluent API on top of any logging library, JSON or pretty rendering, HTTP shipping, OpenTelemetry, or your own transport. Stable API; SemVer applies from this point forward. See [Getting Started](/getting-started).

`v1.1.0`:

- The `fmtlog` plugin moved from `go.loglayer.dev/fmtlog` to `go.loglayer.dev/plugins/fmtlog` for consistency with every other plugin. Update imports:

  ```diff
  - import "go.loglayer.dev/fmtlog"
  + import "go.loglayer.dev/plugins/fmtlog"
  ```

  Technically a SemVer-breaking change (the old import path is gone), but ships as a minor bump rather than v2.0.0 — the v1.0.x release went out with no known users yet, and the post-split module layout (below) means future breaking changes in any single sub-module stay local to that sub-module's tag namespace.

All packages:

- Every transport (`transports/*`), plugin (`plugins/*`), and integration (`integrations/*`) now ships as its own independently-versioned Go module. Import paths are unchanged; you may need a `go mod tidy` to pick up the new sub-module entries. Future breaking changes in any one sub-package bump only that sub-module's major version, leaving the main `go.loglayer.dev` import path stable on v1. Full module list in [`.release-please-manifest.json`](https://github.com/loglayer/loglayer-go/blob/main/.release-please-manifest.json).
