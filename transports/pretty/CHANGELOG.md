# Changelog

All notable changes to `go.loglayer.dev/transports/pretty` are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).
Tags use the prefixed form `transports/pretty/v<X.Y.Z>` so this module
versions independently of the framework core. Maintained
automatically by [monorel](https://monorel.disaresta.com) from
`.changeset/*.md` files.

## [2.1.0] - 2026-05-03

### Minor Changes

- Add `loglayer.Multiline(lines ...any)` for authoring multi-line message
  content that survives terminal-renderer sanitization. The wrapper is
  honored only as a positional message argument; values placed inside
  `WithFields(...)` or `WithMetadata(...)` are still sanitized to a single
  line in terminal transports (JSON sinks serialize via `MarshalJSON` to
  the joined string).

  Also fixes a pre-existing bug in `transport.JoinPrefixAndMessages`
  where a `WithPrefix` value was silently dropped when `Messages[0]`
  was not a string (e.g. `log.WithPrefix("X").Info(42)` lost the
  prefix). The prefix now folds in front of the `%v`-formatted first
  message.

  See https://go.loglayer.dev/logging-api/multiline.

## [2.0.1] - 2026-05-03

### Patch Changes

- Republish every module to ship a clean `go.mod` to the Go module proxy.

  The v2.0.0 cascade and the subsequent `transports/cli` v2.1.0 release shipped sub-module `go.mod` files containing dev-only `replace go.loglayer.dev/v2 => ../..` directives and placeholder pseudo-version requires (`v2.0.0-00010101000000-000000000000`). Downstream consumers who depended on any sub-module saw `go mod tidy` 404 on the placeholder.

  monorel v0.9.0 ([disaresta-org/monorel#42](https://github.com/disaresta-org/monorel/pull/42)) added a release-time `go.mod` cleaner that strips the dev-only sibling replaces and pins sibling requires to the planned release version. This release republishes every affected module with the cleaned `go.mod`.

  No API changes. Re-`go get` to pick up the cleaned modules.

## [2.0.0] - 2026-05-02

### Major Changes

- **Breaking: import paths bump to `/v2`.** The loglayer core no longer folds the `WithPrefix` value into `Messages[0]`; the prefix flows through `TransportParams.Prefix` and each transport renders it. Built-in transports preserve their prior user-visible output via the new `transport.JoinPrefixAndMessages` helper. The `cli` transport opts into smart rendering (dim-grey user prefix separate from level color).

  See [Migrating to v2](https://go.loglayer.dev/migrating-to-v2) for the upgrade checklist.

## [1.6.1](https://github.com/loglayer/loglayer-go/compare/transports/pretty/v1.2.0...transports/pretty/v1.6.1) (2026-04-30)


### Miscellaneous Chores

* Bump every module touched by [#28](https://github.com/loglayer/loglayer-go/issues/28) to refresh pkg.go.dev ([#30](https://github.com/loglayer/loglayer-go/issues/30)) ([2ac74a7](https://github.com/loglayer/loglayer-go/commit/2ac74a7a58947f1d4c1c18ff5998b8042b6d1a59))

## [1.2.0](https://github.com/loglayer/loglayer-go/compare/transports/pretty/v1.1.0...transports/pretty/v1.2.0) (2026-04-30)


### Features

* Surface assembly Schema to transports and plugins, add Sentry transport ([#24](https://github.com/loglayer/loglayer-go/issues/24)) ([d35a0d5](https://github.com/loglayer/loglayer-go/commit/d35a0d5146e704d92f65fb208b17daaa4d151891))

## [1.1.0](https://github.com/loglayer/loglayer-go/compare/transports/pretty/v1.0.0...transports/pretty/v1.1.0) (2026-04-29)

No code changes in this module. v1.1.0 was a coordinated tag created alongside the main `go.loglayer.dev` v1.1.0 release; see the [main CHANGELOG](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md#110-2026-04-29) for the post-launch reshuffle that drove the bump.

## 1.0.0 (2026-04-29)


### ⚠ BREAKING CHANGES

* split pretty/http/datadog into modules; per-module README/CHANGELOG
* drop Trace level, swap to goccy/go-json, rewrite structured transport

### Features

* Drop Trace level, swap to goccy/go-json, rewrite structured transport ([adb7114](https://github.com/loglayer/loglayer-go/commit/adb7114871931074b1e0c2b37de5466adf0e3ddc))
* HTTP/Datadog transports, dev tooling, structure cleanup ([feaacc9](https://github.com/loglayer/loglayer-go/commit/feaacc9cae99581c836edafe70bd5738fdbc2e06))
* Initial v0.1.0 implementation ([bca353c](https://github.com/loglayer/loglayer-go/commit/bca353cf965645d46360cfb3b6f5e43bb4e135d0))
* Optional auto-generated IDs for plugins and transports ([b6fc05a](https://github.com/loglayer/loglayer-go/commit/b6fc05a6c2d4a03359035c7f0aabbf9f77fca8d3))
* Split pretty/http/datadog into modules; per-module README/CHANGELOG ([852d363](https://github.com/loglayer/loglayer-go/commit/852d363829eb622032bf5608010f24755bba2c7c))


### Bug Fixes

* **security:** Cycle-safe Cloner, header sanitization, secret redaction + on-prem Datadog URL ([057c787](https://github.com/loglayer/loglayer-go/commit/057c787681843e611fdf04a83db64cdeed429d7e))

## [Unreleased] (target: v1.0.0)

Initial release as a separate Go module.

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main/transports/pretty
