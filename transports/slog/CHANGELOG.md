# Changelog

## [2.0.0] - 2026-05-02

### Major Changes

- **Breaking: import paths bump to `/v2`.** The loglayer core no longer folds the `WithPrefix` value into `Messages[0]`; the prefix flows through `TransportParams.Prefix` and each transport renders it. Built-in transports preserve their prior user-visible output via the new `transport.JoinPrefixAndMessages` helper. The `cli` transport opts into smart rendering (dim-grey user prefix separate from level color).

  See [Migrating to v2](https://go.loglayer.dev/migrating-to-v2) for the upgrade checklist.

## [1.6.1](https://github.com/loglayer/loglayer-go/compare/transports/slog/v1.3.0...transports/slog/v1.6.1) (2026-04-30)


### Miscellaneous Chores

* Bump every module touched by [#28](https://github.com/loglayer/loglayer-go/issues/28) to refresh pkg.go.dev ([#30](https://github.com/loglayer/loglayer-go/issues/30)) ([2ac74a7](https://github.com/loglayer/loglayer-go/commit/2ac74a7a58947f1d4c1c18ff5998b8042b6d1a59))

## [1.3.0](https://github.com/loglayer/loglayer-go/compare/transports/slog/v1.2.0...transports/slog/v1.3.0) (2026-04-30)


### Features

* Surface assembly Schema to transports and plugins, add Sentry transport ([#24](https://github.com/loglayer/loglayer-go/issues/24)) ([d35a0d5](https://github.com/loglayer/loglayer-go/commit/d35a0d5146e704d92f65fb208b17daaa4d151891))

## [1.2.0](https://github.com/loglayer/loglayer-go/compare/transports/slog/v1.1.0...transports/slog/v1.2.0) (2026-04-29)


### Features

* Cover Trace and Panic levels across transports and contract tests ([d511c7e](https://github.com/loglayer/loglayer-go/commit/d511c7eac484aadc3d3155876e497381b38f75e0))

## [1.1.0](https://github.com/loglayer/loglayer-go/compare/transports/slog/v1.0.0...transports/slog/v1.1.0) (2026-04-29)

No code changes in this module. v1.1.0 was a coordinated tag created alongside the main `go.loglayer.dev` v1.1.0 release; see the [main CHANGELOG](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md#110-2026-04-29) for the post-launch reshuffle that drove the bump.
