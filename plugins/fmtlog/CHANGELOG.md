# Changelog

## [2.0.0] - 2026-05-02

### Major Changes

- **Breaking: import paths bump to `/v2`.** The loglayer core no longer folds the `WithPrefix` value into `Messages[0]`; the prefix flows through `TransportParams.Prefix` and each transport renders it. Built-in transports preserve their prior user-visible output via the new `transport.JoinPrefixAndMessages` helper. The `cli` transport opts into smart rendering (dim-grey user prefix separate from level color).

  See [Migrating to v2](https://go.loglayer.dev/migrating-to-v2) for the upgrade checklist.

## [1.6.1](https://github.com/loglayer/loglayer-go/compare/plugins/fmtlog/v1.1.1...plugins/fmtlog/v1.6.1) (2026-04-30)


### Miscellaneous Chores

* Bump every module touched by [#28](https://github.com/loglayer/loglayer-go/issues/28) to refresh pkg.go.dev ([#30](https://github.com/loglayer/loglayer-go/issues/30)) ([2ac74a7](https://github.com/loglayer/loglayer-go/commit/2ac74a7a58947f1d4c1c18ff5998b8042b6d1a59))

## [1.1.1](https://github.com/loglayer/loglayer-go/compare/plugins/fmtlog/v1.1.0...plugins/fmtlog/v1.1.1) (2026-04-29)


### Code Refactoring

* Split the last four bundled packages so go.loglayer.dev hosts only the framework core ([#13](https://github.com/loglayer/loglayer-go/issues/13)) ([feef8a6](https://github.com/loglayer/loglayer-go/commit/feef8a6cc7b2350b802c43fc78b336037d4a5ea0))

## [1.1.0](https://github.com/loglayer/loglayer-go/compare/plugins/fmtlog/v1.0.0...plugins/fmtlog/v1.1.0) (2026-04-29)

No code changes in this module. v1.1.0 was a coordinated tag created alongside the main `go.loglayer.dev` v1.1.0 release; see the [main CHANGELOG](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md#110-2026-04-29) for the post-launch reshuffle that drove the bump.
