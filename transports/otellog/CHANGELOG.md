# Changelog

All notable changes to `go.loglayer.dev/transports/otellog` are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).

Releases are managed by [Release Please](https://github.com/googleapis/release-please)
from conventional commits scoped to `transports/otellog`. Tags use the
prefixed form `transports/otellog/v<X.Y.Z>` so this module versions
independently of the framework core.

## [1.1.0](https://github.com/loglayer/loglayer-go/compare/transports/otellog/v1.0.1...transports/otellog/v1.1.0) (2026-04-29)


### Miscellaneous Chores

* **release-please:** Force next main release to v1.1.0 ([bbcd772](https://github.com/loglayer/loglayer-go/commit/bbcd77202f70ccd4f4a6f3bbe70f8dc91b4ea695))

## [1.0.1](https://github.com/loglayer/loglayer-go/compare/transports/otellog-v1.0.0...transports/otellog-v1.0.1) (2026-04-29)


### Bug Fixes

* **otel:** Repair stale livetests ([de703c7](https://github.com/loglayer/loglayer-go/commit/de703c7f1e7131e018ccc5f80f5bfe1a451c1abc))

## 1.0.0 (2026-04-29)


### ⚠ BREAKING CHANGES

* split pretty/http/datadog into modules; per-module README/CHANGELOG
* split heavy-dep wrapper transports into sub-modules
* drop Trace level, swap to goccy/go-json, rewrite structured transport

### Features

* Drop Trace level, swap to goccy/go-json, rewrite structured transport ([adb7114](https://github.com/loglayer/loglayer-go/commit/adb7114871931074b1e0c2b37de5466adf0e3ddc))
* **otel:** Logs transport, trace plugin, livetests against real SDKs ([84d8e18](https://github.com/loglayer/loglayer-go/commit/84d8e189102a7d6c037894d5ccc0bd2405e5785a))
* Split heavy-dep wrapper transports into sub-modules ([aef6053](https://github.com/loglayer/loglayer-go/commit/aef6053a1e80c646eb2c666fd0f893b98a66e883))
* Split pretty/http/datadog into modules; per-module README/CHANGELOG ([852d363](https://github.com/loglayer/loglayer-go/commit/852d363829eb622032bf5608010f24755bba2c7c))

## [Unreleased] (target: v1.0.0)

Initial release as a separate Go module. Splits out of the main module
because the OpenTelemetry SDK's Go floor (1.25+) would otherwise bind
the entire framework.

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main/transports/otellog
