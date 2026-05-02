# Changelog

All notable changes to `go.loglayer.dev/transports/lumberjack` are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).
Tags use the prefixed form `transports/lumberjack/v<X.Y.Z>` so this module
versions independently of the framework core. Maintained
automatically by [monorel](https://monorel.disaresta.com) from
`.changeset/*.md` files.

## [2.0.0] - 2026-05-02

### Major Changes

- **Breaking: import paths bump to `/v2`.** The loglayer core no longer folds the `WithPrefix` value into `Messages[0]`; the prefix flows through `TransportParams.Prefix` and each transport renders it. Built-in transports preserve their prior user-visible output via the new `transport.JoinPrefixAndMessages` helper. The `cli` transport opts into smart rendering (dim-grey user prefix separate from level color).

  See [Migrating to v2](https://go.loglayer.dev/migrating-to-v2) for the upgrade checklist.

## [1.6.1](https://github.com/loglayer/loglayer-go/compare/transports/lumberjack/v1.1.0...transports/lumberjack/v1.6.1) (2026-04-30)


### Miscellaneous Chores

* Bump every module touched by [#28](https://github.com/loglayer/loglayer-go/issues/28) to refresh pkg.go.dev ([#30](https://github.com/loglayer/loglayer-go/issues/30)) ([2ac74a7](https://github.com/loglayer/loglayer-go/commit/2ac74a7a58947f1d4c1c18ff5998b8042b6d1a59))

## [1.1.0](https://github.com/loglayer/loglayer-go/compare/transports/lumberjack/v1.0.0...transports/lumberjack/v1.1.0) (2026-04-29)


### Features

* **transports/lumberjack:** Add rotating-file JSON transport ([#16](https://github.com/loglayer/loglayer-go/issues/16)) ([fa7bef0](https://github.com/loglayer/loglayer-go/commit/fa7bef051e0221bb4c3bbb0612bdffa96aeb6869))


### Miscellaneous Chores

* **release-please:** Force next main release to v1.1.0 ([bbcd772](https://github.com/loglayer/loglayer-go/commit/bbcd77202f70ccd4f4a6f3bbe70f8dc91b4ea695))

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
