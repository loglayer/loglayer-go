---
title: Pretty Transport
description: Color-coded, theme-aware terminal output with three view modes.
---

# Pretty Transport

The `pretty` transport renders log entries with ANSI color, theme support, and three view modes (inline, message-only, expanded). Inspired by loglayer's [simple-pretty-terminal](https://loglayer.dev/transports/simple-pretty-terminal).

**This is the recommended transport for local development and any human-readable terminal output.** For production logging, switch to [structured](/transports/structured), [zerolog](/transports/zerolog), or [zap](/transports/zap).

```sh
go get go.loglayer.dev/transports/pretty
```

This transport pulls in `github.com/fatih/color` for ANSI handling.

## Basic Usage

```go
import (
    "go.loglayer.dev"
    "go.loglayer.dev/transports/pretty"
)

log := loglayer.New(loglayer.Config{
    Transport: pretty.New(pretty.Config{}),
})

log.WithMetadata(loglayer.Metadata{"user": "alice", "n": 42}).Info("served")
```

```
12:34:56.789 ▶ INFO served n=42 user=alice
```

(With colors applied by the default Moonlight theme.)

## View Modes

Three rendering modes via `Config.ViewMode`:

### Inline (default)

Single-line output with structured data appended as `key=value`. Best for high-throughput dev logs.

```go
pretty.New(pretty.Config{ViewMode: pretty.ViewModeInline})
```

```
12:34:56.789 ▶ INFO served user=alice n=42
```

### Message-only

Just timestamp, level, and message. Structured data is dropped from output. Useful when you want a clean stream and don't care about per-call payloads.

```go
pretty.New(pretty.Config{ViewMode: pretty.ViewModeMessageOnly})
```

```
12:34:56.789 ▶ INFO served
```

### Expanded

Header on the first line, then YAML-like indented data underneath. Best when payloads are deep or wide:

```go
pretty.New(pretty.Config{ViewMode: pretty.ViewModeExpanded})
```

```
12:34:56.789 ▶ INFO served
  user: alice
  request:
    method: POST
    path: /users
  items:
    - first
    - second
```

## Themes

Five themes ship with the package, mirroring the upstream JS palette:

| Theme         | Best for             |
|---------------|----------------------|
| `Moonlight()` | Dark terminals (default) |
| `Sunlight()`  | Light terminals      |
| `Neon()`      | Dark + high contrast |
| `Nature()`    | Light, organic palette |
| `Pastel()`    | Soft, low-strain     |

```go
pretty.New(pretty.Config{Theme: pretty.Neon()})
```

### Custom Themes

A `*pretty.Theme` is just a struct of `Style` functions (`func(string) string`). Build one with `color.RGB(...)` or any other color library:

```go
import "github.com/fatih/color"

theme := &pretty.Theme{
    Trace:     color.New(color.FgHiBlack).SprintFunc(),
    Debug:     color.New(color.FgCyan).SprintFunc(),
    Info:      color.New(color.FgGreen).SprintFunc(),
    Warn:      color.New(color.FgYellow).SprintFunc(),
    Error:     color.New(color.FgRed).SprintFunc(),
    Fatal:     color.New(color.BgRed, color.FgWhite).SprintFunc(),
    Timestamp: color.New(color.Faint).SprintFunc(),
    LogID:     color.New(color.Faint).SprintFunc(),
    DataKey:   color.New(color.FgCyan).SprintFunc(),
    DataValue: color.New(color.FgWhite).SprintFunc(),
}

pretty.New(pretty.Config{Theme: theme})
```

(Note: `color.New(...).SprintFunc()` returns `func(...any) string`, but `Style` is `func(string) string`. Wrap in `func(s string) string { return c.Sprint(s) }` if needed.)

## Config

```go
type Config struct {
    transport.BaseConfig

    ViewMode        ViewMode               // default: ViewModeInline
    Theme           *Theme                 // default: Moonlight()
    NoColor         bool                   // disable ANSI escape codes
    ShowLogID       bool                   // emit a per-entry [hex-id]
    TimestampFormat string                 // Go time format; default "15:04:05.000"
    TimestampFn     func(time.Time) string // overrides TimestampFormat
    MaxInlineDepth  int                    // default: 4
    Writer          io.Writer              // default: os.Stdout
}
```

### NoColor

Set this when piping to a file, sending to CI, or running in a container that doesn't honor ANSI:

```go
pretty.New(pretty.Config{NoColor: true})
```

`fatih/color` also auto-disables when stdout isn't a TTY, so in most CI scenarios you don't need to set this manually.

### ShowLogID

Each entry gets a short pseudo-random identifier prefixed to the message. Useful for cross-referencing multi-line expanded output:

```go
pretty.New(pretty.Config{ShowLogID: true})
```

```
12:34:56.789 ▶ INFO [00001a] served
12:34:56.790 ▶ WARN [00001b] retry
```

### Timestamps

Default is `15:04:05.000` (HH:MM:SS.mmm). Override with `TimestampFormat` (any Go time layout) or `TimestampFn` for full control:

```go
pretty.New(pretty.Config{
    TimestampFormat: "2006-01-02 15:04:05",
})

pretty.New(pretty.Config{
    TimestampFn: func(t time.Time) string {
        return strconv.FormatInt(t.Unix(), 10)
    },
})
```

### MaxInlineDepth

In inline mode, nested objects deeper than this collapse to `{...}` to keep lines short:

```go
pretty.New(pretty.Config{MaxInlineDepth: 2})

log.WithMetadata(loglayer.Metadata{
    "shallow": "ok",
    "deep":    loglayer.Metadata{"nested": loglayer.Metadata{"more": "stuff"}},
}).Info("depth")
// 12:34:56.789 ▶ INFO depth deep={nested={...}} shallow=ok
```

Doesn't affect expanded mode; it always shows the full tree.

## Metadata Handling

- **Maps** merge at the root, alongside fields and error fields.
- **Structs** are JSON-roundtripped into a map (so `json:"foo"` tags determine the rendered key) and merged at the root.
- **Scalars / unknown types** fall back to `_metadata` as the key.

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")
// 12:34:56.789 ▶ INFO user id=7 name=Alice
```

## Fatal Behavior

<!--@include: ./_partials/fatal-passthrough.md-->

## Threat Model: Use for Terminals, Not for Pipelines

Pretty is a terminal renderer. It writes ANSI color codes for level chevrons, keys, and timestamps directly to the writer (defaults to stdout). The message string is sanitized for control characters before output, but **field/metadata values** in pretty's rendered output pass through to the terminal in their typed form, including any ANSI escape sequences they happen to contain.

Concretely: an attacker who can place a value in a logged field could include `\x1b[31m...` in that value, which a viewer's terminal would interpret as red text. They could also use `\r` to overwrite previous output or `\x1b[2J` to clear the screen.

If your service ships log lines to a viewer who can be tricked by terminal control sequences, **use [`structured`](/transports/structured) instead**. Structured emits JSON; encoding/json escapes all control characters by default, so the threat model is closed. Pretty's contract is "for the developer's terminal during local dev"; that's also where it's useful.

## Combining with Other Transports

Use it locally alongside a structured transport that ships to disk or a service:

```go
loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        pretty.New(pretty.Config{
            BaseConfig: transport.BaseConfig{ID: "console"},
        }),
        structured.New(structured.Config{
            BaseConfig: transport.BaseConfig{
                ID:    "ship",
                Level: loglayer.LogLevelWarn,
            },
            Writer: logFile,
        }),
    },
})
```
