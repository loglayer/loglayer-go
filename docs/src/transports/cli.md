---
title: CLI Transport
description: "Plain-text output tuned for command-line apps: short level prefixes, stdout / stderr routing, TTY-detected color, no timestamps."
---

# CLI Transport

<ModuleBadges path="transports/cli" />

The `cli` transport renders log entries as plain user-facing CLI output. The closest cousin among the built-in transports is [console](/transports/console), but `cli` is opinionated for command-line ergonomics rather than diagnostic logging:

- **No timestamp, no log-id, no level label embedded in info / debug output.** The message is printed as-is.
- **Short cargo / eslint-style prefixes for warn / error / fatal**: `warning: `, `error: `, `fatal: `.
- **Stdout for info / debug; stderr for warn / error / fatal / panic.**
- **TTY-detected color.** Pipe to a file or another process and ANSI escapes auto-disable. Override via `Config.Color`.
- **Fields and metadata are dropped by default.** CLI users don't want `key=value` noise on user-facing output. Set `ShowFields: true` when wiring `-vv` / `--debug` to a verbose mode.
- **Table rendering for slice metadata.** Pass `[]loglayer.Metadata`, `[]SomeStruct`, or any other slice of map-shaped or struct-shaped values to `WithMetadata` / `MetadataOnly` and the transport renders a tabwriter-aligned table after the message. Same call site emits a proper JSON array when paired with the [structured](/transports/structured) transport. See [Table Rendering for Slice-of-Map Metadata](#table-rendering-for-slice-of-map-metadata) below.

```sh
go get go.loglayer.dev/transports/cli/v2
```

## Basic Usage

```go
import (
    "go.loglayer.dev/v2"
    cli "go.loglayer.dev/transports/cli/v2"
)

log := loglayer.New(loglayer.Config{
    Transport: cli.New(cli.Config{}),
})

log.Info("Applied 1 release(s) at f5f6a9a:")
log.Warn("running on stale credentials")
log.Error("connection refused")
```

Output (stdout + stderr merged for illustration):

```
Applied 1 release(s) at f5f6a9a:
warning: running on stale credentials
error: connection refused
```

`log.Info` writes to stdout with no decoration. `log.Warn` and `log.Error` write to stderr with their respective short prefixes. When stdout is a TTY, the warn line is yellow and the error line is red.

## Config

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Stdout` | `io.Writer` | `os.Stdout` | Override for the info / debug / trace stream. |
| `Stderr` | `io.Writer` | `os.Stderr` | Override for the warn / error / fatal / panic stream. |
| `Color` | `ColorMode` | `ColorAuto` | One of `ColorAuto` (color when stdout is a TTY), `ColorAlways`, or `ColorNever`. Wire your CLI's `--color` flag through this. |
| `ShowFields` | `bool` | `false` | When true, append `key=value` pairs (logfmt) after the message. Useful for `-vv` / `--debug` verbosity modes. |
| `LevelPrefix` | `map[loglayer.LogLevel]string` | see below | Override the per-level prefix. Missing entries fall back to defaults. Set an entry to `""` to suppress the default prefix for that level only. |
| `DisableLevelPrefix` | `bool` | `false` | Master switch: when true, every level's prefix is suppressed regardless of `LevelPrefix`. Use when the host CLI already renders its own urgency markers. |
| `LevelColor` | `map[loglayer.LogLevel]*color.Color` | see below | Override the per-level color. Missing entries fall back to defaults. Set an entry to `nil` to render that level without color while keeping other defaults. Use a custom `*color.Color` (from `fatih/color`) to rebrand. |

Default `LevelPrefix`:

| Level | Prefix |
|-------|--------|
| Trace | `""` |
| Debug | `"debug: "` |
| Info | `""` |
| Warn | `"warning: "` |
| Error | `"error: "` |
| Fatal | `"fatal: "` |
| Panic | `"panic: "` |

Default `LevelColor`:

| Level | Color |
|-------|-------|
| Trace | dim grey (`color.FgHiBlack`) |
| Debug | dim grey (`color.FgHiBlack`) |
| Info | none |
| Warn | yellow (`color.FgYellow`) |
| Error | red (`color.FgRed`) |
| Fatal | bold red (`color.FgRed`, `color.Bold`) |
| Panic | bold red (`color.FgRed`, `color.Bold`) |

## Table Rendering for Slice-of-Map Metadata

When the value passed to `WithMetadata` (or `MetadataOnly`) is a slice of map-shaped entries (`[]loglayer.Metadata`, `[]map[string]any`, or `[]any` whose every element is a map), the transport renders a tabwriter-aligned table after the message. Same call site, two appropriate renderings: CLI sees the table, [structured](/transports/structured) sees the proper JSON array.

```go
log.WithMetadata([]loglayer.Metadata{
    {"package": "transports/foo", "from": "v1.5.0", "to": "v1.6.0"},
    {"package": "transports/bar", "from": "v0.2.0", "to": "v1.0.0"},
}).Info("Plan:")
```

```
Plan:
FROM    PACKAGE         TO
v1.5.0  transports/foo  v1.6.0
v0.2.0  transports/bar  v1.0.0
```

Rules:

- **Column order**: union of keys across all rows, sorted lexicographically. Stable across runs.
- **Column header**: each key uppercased.
- **Missing values**: empty cell.
- **Padding**: two spaces between columns (matches `gh`, `kubectl get`, `cargo`).
- **Slice-of-struct** (`[]MyType` or `[]*MyType`) is also accepted. Each element is JSON-roundtripped, so JSON struct tags become column headers. Given a struct field tagged `json:"package"`, the rendered table uses `PACKAGE` as that column's header.
- **Heterogeneous slices** (a non-map element mixed in): the transport bails out and renders the message alone, no table.
- **Empty slices**: no table, no extra newlines.
- **Single-map metadata** (`WithMetadata(loglayer.Metadata{...})`): the existing logfmt path applies if `ShowFields` is true. Table mode is opt-in via array shape only.
- **Compatible with `MetadataOnly`**: `log.MetadataOnly([]loglayer.Metadata{...})` emits just the table, no leading blank line.

When `ShowFields` is also true, table rendering takes precedence over logfmt for that entry.

When the entry's level is warn / error / fatal, the headline (prefix + message) is colored, but the table body renders neutral. Tables are data, not warnings; tinting the rows would be visually misleading.

## Using `WithPrefix`

The cli transport reads `params.Prefix` directly and renders it as a third visual layer between the level prefix and the message body, in dim grey. Each piece gets its own color treatment so caller-context and urgency stay visually distinct:

```go
log := loglayer.New(...).WithPrefix("[auth]")
log.Info("starting")    // → "[auth] starting"            (prefix dim grey)
log.Warn("retrying")    // → "warning: [auth] retrying"   (level yellow, prefix grey, body yellow)
log.Error("failed")     // → "error: [auth] failed"       (level red, prefix grey, body red)
```

The level prefix and message body share the level color (yellow / red / etc.); the user prefix gets `color.FgHiBlack` (dim grey) regardless of level. The visual layering reads as "this is a warning. [auth context] retrying": three signals stacked rather than blended.

If you want monochrome rendering, set `Color: ColorNever` to drop all color (the user prefix and the level prefix both render as plain text).

## Verbose Mode (`-v` / `--debug`)

The standard CLI shape is to wire `-v` flags to loglayer's level state and `-vv` (or `--debug`) to also enable `ShowFields`. `verbosity` here is the count of `-v` flags: `-v` = 1, `-vv` = 2.

```go
import (
    "go.loglayer.dev/v2"
    cli "go.loglayer.dev/transports/cli/v2"
)

func newLogger(verbosity int) *loglayer.LogLayer {
    log := loglayer.New(loglayer.Config{
        Transport: cli.New(cli.Config{
            ShowFields: verbosity >= 2,
        }),
    })
    switch {
    case verbosity >= 2:
        log.SetLevel(loglayer.LogLevelDebug)
    case verbosity >= 1:
        log.SetLevel(loglayer.LogLevelInfo)
    default:
        log.SetLevel(loglayer.LogLevelWarn)
    }
    return log
}
```

`SetLevel` is concurrency-safe via an atomic.Uint32, so a `--quiet` flag can lower the level after construction without re-wiring transports.

## `--color=auto|always|never`

Most CLI conventions accept `--color=auto` (default), `--color=always`, and `--color=never`. Map them directly:

```go
var color cli.ColorMode
switch flagColor {
case "auto", "":
    color = cli.ColorAuto
case "always":
    color = cli.ColorAlways
case "never":
    color = cli.ColorNever
default:
    return fmt.Errorf("invalid --color=%q: want auto|always|never", flagColor)
}
```

`ColorAuto` checks whether the resolved stdout is a terminal at construction time, and that decision is pinned for the lifetime of the transport. If your CLI is invoked from a wrapper that pipes stdout, color disables automatically.

Note that the TTY check is against `Stdout`, not `Stderr`: piping stdout to a file disables color on stderr-bound warn / error / fatal lines too. This matches how `gh`, `kubectl`, and most modern CLIs behave; the operator either wants color everywhere or nowhere, not a half-and-half mix.

## Recommended Plugin Pairings

### `fmtlog` for `fmt.Sprintf` semantics

Without a plugin, multi-argument log calls are space-joined: `log.Info("count:", n)` renders as `"count: 1234"`. CLI output usually wants format-string semantics:

```go
import "go.loglayer.dev/plugins/fmtlog/v2"

log := loglayer.New(loglayer.Config{
    Transport: cli.New(cli.Config{}),
    Plugins:   []loglayer.Plugin{fmtlog.New()},
})

log.Info("Applied %d release(s) at %s:", count, sha)
log.Error("connecting to %s: %v", host, err)
```

The plugin registers a single MessageHook that rewrites `[]any{format, args...}` to `[]any{fmt.Sprintf(format, args...)}`. Zero hot-path cost when a call has a single message; one Sprintf when there are extras. See [fmtlog](https://pkg.go.dev/go.loglayer.dev/plugins/fmtlog/v2) for the full API.

### `redact` for token scrubbing

If your CLI ever logs values that might include `GITHUB_TOKEN`, `GITLAB_TOKEN`, or other secrets via `WithMetadata`, pair with the [redact](https://pkg.go.dev/go.loglayer.dev/plugins/redact/v2) plugin. ANSI / CRLF sanitization is already on the table-cell and logfmt-value paths in this transport; the redact plugin closes the value-content side (token patterns, key allow / deny lists).

## Switching to JSON for `--json`

For machine-readable output, swap `cli` for [structured](/transports/structured) without changing call sites:

```go
import (
    "go.loglayer.dev/v2"
    cli "go.loglayer.dev/transports/cli/v2"
    "go.loglayer.dev/transports/structured/v2"
)

var t loglayer.Transport
if jsonOutput {
    t = structured.New(structured.Config{})
} else {
    t = cli.New(cli.Config{})
}
log := loglayer.New(loglayer.Config{Transport: t})
```

Same `log.Info(...)` / `log.Warn(...)` / `WithFields(...)` chain; the transport decides the output shape. JSON callers get every field; CLI callers get the prefix-and-message form.

## Fatal Behavior

The transport writes the entry; loglayer's core decides whether `os.Exit(1)` is called via `Config.DisableFatalExit`. Fatal lines render with the `fatal: ` prefix and bold red color (when ANSI is enabled).

## Metadata Handling

Metadata is dropped by default (`ShowFields: false`). When `ShowFields: true`, the transport calls `transport.MergeFieldsAndMetadata` and renders the merged map as space-separated `key=value` pairs in sorted-key order. Values containing spaces, equals signs, quotes, or control characters are quoted via `%q`.

## GetLoggerInstance

Returns `nil`. The CLI transport has no underlying logger library to expose.
