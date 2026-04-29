# Changelog

All notable changes to `go.loglayer.dev/transports/lumberjack` are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).

Releases are managed by [Release Please](https://github.com/googleapis/release-please)
from conventional commits scoped to `transports/lumberjack`. Tags use the
prefixed form `transports/lumberjack/v<X.Y.Z>` so this module versions
independently of the framework core.

## [Unreleased] (target: v1.0.0)

Initial release as a separate Go module.

### Added

- File transport that writes one JSON object per log entry to a rotating file. Rotation is delegated to [lumberjack.v2](https://github.com/natefinch/lumberjack): size-triggered rollover, configurable backup retention, age-based cleanup, and optional gzip compression. Render path matches `transports/structured` so the on-disk format is identical to the structured transport's stdout output.
- `New` / `Build` constructor pair; `Build` returns `ErrFilenameRequired` instead of panicking when `Config.Filename` is empty.
- `Config.OnError` callback for write/rotate failures. Default implementation writes a one-line message to `os.Stderr`; pass a custom callback to plumb failures into a logger, alerting pipeline, or metrics counter.
- `Rotate()` to force an immediate roll-over (e.g. from a SIGHUP handler).
- `Close()` to release the file handle. Post-close log calls are silently dropped so lumberjack's lazy-reopen does not revive the file.
- `GetLoggerInstance()` returns the underlying `*lumberjack.Logger` for direct operational access.

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main/transports/lumberjack
