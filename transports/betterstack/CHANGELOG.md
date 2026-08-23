# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-08-23

### Major Changes

- **Breaking: module paths bump to the next major.** These transports, plugins, and integrations now depend on `go.loglayer.dev/v3` and re-export v3 core types (or sibling v3 types) in their public API. Per Go convention, each module's path moves to its next major (`/v2` → `/v3`, or unversioned → `/v2`); consumers update imports and `go get` lines.

  The `structured` transport also sanitizes ANSI escape sequences, bidi overrides, and CR/LF at the top level of every entry and omits the `msg` key when the message is empty. See the migration guide at https://go.loglayer.dev/migrating.

## [1.0.0] - 2026-05-11

### Major Changes

- Initial release.

