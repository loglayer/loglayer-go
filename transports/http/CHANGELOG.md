# Changelog

All notable changes to `go.loglayer.dev/transports/http` are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).

Releases are managed by [Release Please](https://github.com/googleapis/release-please)
from conventional commits scoped to `transports/http`. Tags use the
prefixed form `transports/http/v<X.Y.Z>` so this module versions
independently of the framework core.

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
