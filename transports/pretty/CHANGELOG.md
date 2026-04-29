# Changelog

All notable changes to `go.loglayer.dev/transports/pretty` are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).

Releases are managed by [Release Please](https://github.com/googleapis/release-please)
from conventional commits scoped to `transports/pretty`. Tags use the
prefixed form `transports/pretty/v<X.Y.Z>` so this module versions
independently of the framework core.

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
