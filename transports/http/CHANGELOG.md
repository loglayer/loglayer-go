# Changelog

All notable changes to `go.loglayer.dev/transports/http` are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).
Tags use the prefixed form `transports/http/v<X.Y.Z>` so this module
versions independently of the framework core. Maintained
automatically by [monorel](https://monorel.disaresta.com) from
`.changeset/*.md` files.

## [2.1.0] - 2026-05-03

### Minor Changes

- Add `Config.String()` that redacts `Headers` values so an accidental `log.Info(cfg)` or `fmt.Sprintf("%v", cfg)` can't leak credentials passed via `Authorization` / `X-API-Key` / similar headers. Header keys stay visible for debuggability. Mirrors the redaction shape already used by `transports/datadog`.

  `defaultCheckRedirect` now compares hosts case-insensitively, so legitimate same-host redirects with mixed-case URLs (`Example.COM` → `example.com`) aren't refused. Cross-host refusal still applies; ports are still compared exactly.

  New `Config.ShutdownTimeout` (default 5s) bounds how long `Close` waits for in-flight requests to finish during shutdown. When the timeout elapses, the worker's outbound HTTP requests are cancelled via context so `Close` can return even if the endpoint is wedged; previously a stuck endpoint could pin `Close` for up to the per-request `Client.Timeout` (30s default), and the parent `flushTransports`'s 5s timeout would leak the close goroutine. Outbound requests are now built via `http.NewRequestWithContext`.

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

## [1.6.1](https://github.com/loglayer/loglayer-go/compare/transports/http/v1.3.0...transports/http/v1.6.1) (2026-04-30)


### Miscellaneous Chores

* Bump every module touched by [#28](https://github.com/loglayer/loglayer-go/issues/28) to refresh pkg.go.dev ([#30](https://github.com/loglayer/loglayer-go/issues/30)) ([2ac74a7](https://github.com/loglayer/loglayer-go/commit/2ac74a7a58947f1d4c1c18ff5998b8042b6d1a59))

## [1.3.0](https://github.com/loglayer/loglayer-go/compare/transports/http/v1.2.0...transports/http/v1.3.0) (2026-04-30)


### Features

* Surface assembly Schema to transports and plugins, add Sentry transport ([#24](https://github.com/loglayer/loglayer-go/issues/24)) ([d35a0d5](https://github.com/loglayer/loglayer-go/commit/d35a0d5146e704d92f65fb208b17daaa4d151891))

## [1.2.0](https://github.com/loglayer/loglayer-go/compare/transports/http/v1.1.0...transports/http/v1.2.0) (2026-04-30)


### Features

* Surface entry groups to transports and dispatch-time plugin hooks ([#22](https://github.com/loglayer/loglayer-go/issues/22)) ([6db2209](https://github.com/loglayer/loglayer-go/commit/6db2209614bfc1ad02b22502a52e409ed130b2b8))

## [1.1.0](https://github.com/loglayer/loglayer-go/compare/transports/http/v1.0.0...transports/http/v1.1.0) (2026-04-29)

No code changes in this module. v1.1.0 was a coordinated tag created alongside the main `go.loglayer.dev` v1.1.0 release; see the [main CHANGELOG](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md#110-2026-04-29) for the post-launch reshuffle that drove the bump.

## 1.0.0 (2026-04-29)


### ⚠ BREAKING CHANGES

* split pretty/http/datadog into modules; per-module README/CHANGELOG
* drop Trace level, swap to goccy/go-json, rewrite structured transport

### Features

* Drop Trace level, swap to goccy/go-json, rewrite structured transport ([adb7114](https://github.com/loglayer/loglayer-go/commit/adb7114871931074b1e0c2b37de5466adf0e3ddc))
* HTTP/Datadog transports, dev tooling, structure cleanup ([feaacc9](https://github.com/loglayer/loglayer-go/commit/feaacc9cae99581c836edafe70bd5738fdbc2e06))
* Slog.Handler integration, security hardening, doc lead rewrite ([bcd4a1e](https://github.com/loglayer/loglayer-go/commit/bcd4a1e8679a28cb2454b78c1c04a198238d7a8f))
* Split pretty/http/datadog into modules; per-module README/CHANGELOG ([852d363](https://github.com/loglayer/loglayer-go/commit/852d363829eb622032bf5608010f24755bba2c7c))


### Bug Fixes

* 100go.co review — substring leaks, http transport TOCTOU/leaks ([ca4b08d](https://github.com/loglayer/loglayer-go/commit/ca4b08d8e2498f2921f5afc6fa9fa6adaa8b9924))
* **loghttp:** Sanitize panic value before logging ([8bf612d](https://github.com/loglayer/loglayer-go/commit/8bf612ddbb66f6072d88bf0793f093a146dab74b))
* **security:** Cycle-safe Cloner, header sanitization, secret redaction + on-prem Datadog URL ([057c787](https://github.com/loglayer/loglayer-go/commit/057c787681843e611fdf04a83db64cdeed429d7e))

## [Unreleased] (target: v1.0.0)

Initial release as a separate Go module.

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main/transports/http
