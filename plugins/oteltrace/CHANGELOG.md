# Changelog

All notable changes to `go.loglayer.dev/plugins/oteltrace` are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).

Releases are managed by [Release Please](https://github.com/googleapis/release-please)
from conventional commits scoped to `plugins/oteltrace`. Tags use the
prefixed form `plugins/oteltrace/v<X.Y.Z>` so this module versions
independently of the framework core.

## [Unreleased] (target: v1.0.0)

Initial release as a separate Go module. Splits out of the main module
because the OpenTelemetry trace API's Go floor would otherwise bind the
entire framework.

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main/plugins/oteltrace
