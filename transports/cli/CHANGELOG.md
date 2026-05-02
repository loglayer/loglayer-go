# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-05-02

### Major Changes

- **Breaking: import paths bump to `/v2`.** The loglayer core no longer folds the `WithPrefix` value into `Messages[0]`; the prefix flows through `TransportParams.Prefix` and each transport renders it. Built-in transports preserve their prior user-visible output via the new `transport.JoinPrefixAndMessages` helper. The `cli` transport opts into smart rendering (dim-grey user prefix separate from level color).

  See [Migrating to v2](https://go.loglayer.dev/migrating-to-v2) for the upgrade checklist.

## [1.0.0] - 2026-05-02

### Major Changes

- Promote to v1.0.0. The transports/cli API has been validated by the planned monorel migration and is ready to be SemVer-committed; v0.1.0 was an unintentional pre-1.0 release that the new `.claude/rules/release-versioning.md` rule guards against in the future.

## [0.1.0] - 2026-05-02

### Minor Changes

- Initial release. New [CLI transport](/transports/cli) tuned for command-line app output rather than diagnostic logging:

  - No timestamp, no log-id, no level label embedded in info / debug output.
  - Short cargo / eslint-style prefixes for warn / error / fatal (`warning: `, `error: `, `fatal: `). Per-level overrides via `LevelPrefix`; master switch via `DisableLevelPrefix` for hosts that supply their own urgency markers.
  - Per-level color overrides via `LevelColor` (rebrand to cyan / magenta / whatever, or set entries to `nil` to render specific levels uncolored).
  - Stdout for info / debug; stderr for warn / error / fatal / panic.
  - TTY-detected color via fatih/color, with `ColorAuto` / `ColorAlways` / `ColorNever` modes for wiring `--color` flags.
  - Fields and metadata dropped by default; `ShowFields: true` appends them in logfmt for `-vv` / `--debug` modes.
  - **Table rendering for slice metadata.** When `WithMetadata` (or `MetadataOnly`) receives `[]loglayer.Metadata`, `[]map[string]any`, `[]any` of maps, `[]SomeStruct`, or `[]*SomeStruct`, the transport renders a tabwriter-aligned table after the message. Struct slices are JSON-roundtripped so JSON tags become column headers. Same call site emits a proper JSON array when paired with the [structured](/transports/structured) transport.

  Pair with the structured transport to get a one-line transport swap when the CLI's `--json` flag is set.

