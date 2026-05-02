---
title: Console Transport
description: Plain-text logs to stdout/stderr in logfmt key=value format.
---

# Console Transport

<ModuleBadges path="transports/console" />

The `console` transport writes log entries to `os.Stdout` (trace, debug, info) or `os.Stderr` (warn, error, fatal, panic) as plain text: the message followed by [logfmt](https://brandur.org/logfmt)-style `key=value` pairs. One line per entry, no colors, no JSON.

::: tip For human-readable dev output, prefer Pretty
The console transport is intentionally minimal. For day-to-day local development you almost certainly want the [Pretty Transport](/transports/pretty): it provides color-coded levels, themes, and three view modes that make multi-field logs easy to scan. Pick `console` only when you specifically need a no-color, no-JSON, logfmt-style writer (e.g. CI logs that humans grep, fixture generation, deliberate plain output).
:::

For production logging, use the [structured](/transports/structured) transport or one of the [logger wrappers](/transports/zerolog).

```sh
go get go.loglayer.dev/transports/console/v2
```

## Basic Usage

```go
import (
    "go.loglayer.dev/v2"
    "go.loglayer.dev/transports/console/v2"
)

log := loglayer.New(loglayer.Config{
    Transport:         console.New(console.Config{}),
    MetadataFieldName: "metadata",
})

log.Info("hello world")
log.WithMetadata(loglayer.Metadata{"id": 42, "user": "alice"}).Info("logged in")
```

```
hello world
logged in metadata="{\"id\":42,\"user\":\"alice\"}"
```

## Output Routing

By default:

- `Trace`, `Debug`, `Info` → `os.Stdout`
- `Warn`, `Error`, `Fatal`, `Panic` → `os.Stderr`

Set `Writer` to override and send everything to a single writer (useful in tests or when redirecting output):

```go
console.New(console.Config{Writer: &buf})
```

## Config

```go
type Config struct {
    transport.BaseConfig

    MessageField string // when set, emit one structured object per entry
    DateField    string // include timestamp under this key
    LevelField   string // include level under this key

    DateFn   func() string                              // override default ISO-8601 timestamp
    LevelFn  func(loglayer.LogLevel) string             // override level string
    MessageFn func(loglayer.TransportParams) string     // format the entire message
    Stringify bool                                       // JSON-encode the structured object

    Writer io.Writer
}
```

### Logfmt Rendering

Fields and metadata render as `key=value` pairs after the message, in sorted-by-key order:

```go
log.WithFields(loglayer.Fields{"requestId": "abc"}).
    WithMetadata(loglayer.Metadata{"status": 200, "bytes": 1024}).
    Info("request served")
// "request served bytes=1024 requestId=abc status=200"
```

Values render based on type:

- **Strings**: bare when safe, quoted when they contain spaces, `=`, `"`, `\`, or control characters. `\"`, `\\`, `\n`, `\r`, `\t` are escaped inside quotes.
- **Numbers and bools**: rendered directly (`id=42`, `enabled=true`).
- **`time.Time`**: RFC3339Nano (`ts=2026-04-26T12:00:00Z`).
- **`error`**: the result of `Error()`, quoted if needed.
- **Anything else (maps, structs, slices)**: JSON-encoded and treated as a quoted string (`obj="{\"a\":1}"`).

Keys are sorted to keep output stable across runs (Go map iteration is non-deterministic).

### Level / Timestamp

By default the line is just message + fields. Add a level or timestamp via `LevelField` / `DateField`:

```go
console.New(console.Config{
    DateField:  "ts",
    LevelField: "level",
})

log.Info("served")
// "served level=info ts=2026-04-26T12:00:00Z"
```

Override the value formatters with `LevelFn` / `DateFn`:

```go
console.New(console.Config{
    LevelField: "lvl",
    LevelFn:    func(l loglayer.LogLevel) string { return strings.ToUpper(l.String()) },
    DateField:  "ts",
    DateFn:     func() string { return time.Now().Format(time.RFC822) },
})
```

### Single-Object Output (`MessageField`)

Setting `MessageField` switches the transport into structured-object mode. Instead of `msg key=value...`, it builds one map containing the message + data + (optionally) timestamp/level fields, and prints just that:

```go
console.New(console.Config{
    MessageField: "msg",
    DateField:    "ts",
    LevelField:   "level",
})

log.Info("structured")
// map[level:info msg:structured ts:2026-04-26T...]
```

Add `Stringify: true` to JSON-encode the object instead:

```go
console.New(console.Config{MessageField: "msg", Stringify: true})
log.Info("json out")
// {"msg":"json out"}
```

If you only want JSON output, prefer the [structured transport](/transports/structured); it's purpose-built for that.

### Custom Message Function

`MessageFn` replaces the default message-then-logfmt assembly with a single string of your choice. It receives the full `TransportParams` so you can format based on the level, fields, or anything else.

```go
console.New(console.Config{
    MessageFn: func(p loglayer.TransportParams) string {
        return fmt.Sprintf("[%s] %s", p.LogLevel, strings.Join(stringifyMessages(p.Messages), " | "))
    },
})
```

When `MessageFn` is set, the logfmt tail still appends after its return value (use it for header formatting, not full takeover).

## Metadata Handling

Map metadata is merged into the same data bag as fields and errors. Struct metadata is JSON-roundtripped into root fields before logfmt rendering. See [Metadata](/logging-api/metadata) for the design.

## Threat Model: Plaintext (Not for Pipelines)

User message strings are sanitized for control characters before output (CR, LF, ESC, Unicode bidi controls, zero-width joiners; see `sanitize.Message`). Field and metadata **string values** are quoted and `\n` / `\r` / `\t` are escaped, so user-controlled string fields can't forge log lines. Non-string values pass through their typed renderer (numbers, bools, JSON encoding for nested), which doesn't introduce control characters.

If your service has untrusted input flowing into a logged field and the resulting log lines are read by a parser sensitive to subtle escape variations, **use [`structured`](/transports/structured) instead**. Structured emits JSON; encoding/json applies a strict, well-specified escaping. Console (like pretty) is for the developer's terminal during local dev.

## Fatal Behavior

<!--@include: ./_partials/fatal-passthrough.md-->
