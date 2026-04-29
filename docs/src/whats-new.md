---
title: What's new in LogLayer for Go
description: Latest features and improvements in LogLayer for Go.
---

# What's new in LogLayer for Go

- [`go.loglayer.dev` Changelog](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md)
- For users coming from the TypeScript [`loglayer`](https://loglayer.dev) library, see [For TypeScript Developers](/for-typescript-developers) for the API mapping.

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
