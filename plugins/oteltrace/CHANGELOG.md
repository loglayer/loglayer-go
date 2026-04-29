# Changelog

All notable changes to `go.loglayer.dev/plugins/oteltrace` are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).

Releases are managed by [Release Please](https://github.com/googleapis/release-please)
from conventional commits scoped to `plugins/oteltrace`. Tags use the
prefixed form `plugins/oteltrace/v<X.Y.Z>` so this module versions
independently of the framework core.

## 1.0.0 (2026-04-29)


### ⚠ BREAKING CHANGES

* split pretty/http/datadog into modules; per-module README/CHANGELOG
* split heavy-dep wrapper transports into sub-modules
* rework Plugin from struct-of-funcs to interface-based

### Features

* **otel:** Logs transport, trace plugin, livetests against real SDKs ([84d8e18](https://github.com/loglayer/loglayer-go/commit/84d8e189102a7d6c037894d5ccc0bd2405e5785a))
* **oteltrace:** Emit W3C trace_state and baggage members ([da40e25](https://github.com/loglayer/loglayer-go/commit/da40e25950942c8dbc2572ca57d5235cb83e5369))
* Rework Plugin from struct-of-funcs to interface-based ([0a95c1a](https://github.com/loglayer/loglayer-go/commit/0a95c1a3445b6cdfdc2669b3474ef092e154d318))
* Split heavy-dep wrapper transports into sub-modules ([aef6053](https://github.com/loglayer/loglayer-go/commit/aef6053a1e80c646eb2c666fd0f893b98a66e883))
* Split pretty/http/datadog into modules; per-module README/CHANGELOG ([852d363](https://github.com/loglayer/loglayer-go/commit/852d363829eb622032bf5608010f24755bba2c7c))

## [Unreleased] (target: v1.0.0)

Initial release as a separate Go module. Splits out of the main module
because the OpenTelemetry trace API's Go floor would otherwise bind the
entire framework.

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main/plugins/oteltrace
