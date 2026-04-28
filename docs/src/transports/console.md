---
title: Console Transport
description: Human-readable log output to stdout/stderr.
---

# Console Transport

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/console.svg)](https://pkg.go.dev/go.loglayer.dev/transports/console) [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=v*&label=go.loglayer.dev)](https://github.com/loglayer/loglayer-go/releases) [![Source](https://img.shields.io/badge/source-github-181717?logo=github)](https://github.com/loglayer/loglayer-go/tree/main/transports/console) [![Changelog](https://img.shields.io/badge/changelog-md-blue)](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md)

The `console` transport writes log entries to `os.Stdout` (info, debug, trace) or `os.Stderr` (warn, error, fatal). Output is formatted with `fmt.Println`-style printing: one line per entry, with structured data appended or prepended as a Go map (e.g. `map[k:v]`).

::: tip For human-readable dev output, prefer Pretty
The console transport is intentionally minimal. For day-to-day local development you almost certainly want the [Pretty Transport](/transports/pretty): it provides color-coded levels, themes, and three view modes that make multi-field logs easy to scan. Pick `console` only when you specifically need a zero-dependency, no-color, raw `fmt`-style writer (e.g. constrained environments, fixture generation, deliberate plain output).
:::

For production logging, use the [structured](/transports/structured) transport or one of the [logger wrappers](/transports/zerolog).

```sh
go get go.loglayer.dev/transports/console
```

## Basic Usage

```go
import (
    "go.loglayer.dev"
    "go.loglayer.dev/transports/console"
)

log := loglayer.New(loglayer.Config{
    Transport: console.New(console.Config{}),
})

log.Info("hello world")
log.WithMetadata(loglayer.Metadata{"k": "v"}).Info("with data")
```

```
hello world
map[k:v] with data
```

## Output Routing

By default:

- `Debug`, `Info` → `os.Stdout`
- `Warn`, `Error`, `Fatal` → `os.Stderr`

Set `Writer` to override and send everything to a single writer (useful in tests or when redirecting output):

```go
console.New(console.Config{Writer: &buf})
```

## Config

```go
type Config struct {
    transport.BaseConfig

    AppendObjectData bool   // append data after messages instead of prepending
    MessageField     string // when set, emit one structured object per entry
    DateField        string // include timestamp under this key
    LevelField       string // include level under this key

    DateFn   func() string                              // override default ISO-8601 timestamp
    LevelFn  func(loglayer.LogLevel) string             // override level string
    MessageFn func(loglayer.TransportParams) string     // format the entire message
    Stringify bool                                       // JSON-encode the structured object

    Writer io.Writer
}
```

### Append vs Prepend

By default the data map is prepended to the message arguments. Set `AppendObjectData: true` to put it after:

```go
console.New(console.Config{AppendObjectData: true})
log.WithMetadata(loglayer.Metadata{"x": 1}).Info("event")
// "event map[x:1]"
```

### Single-Object Output

Setting `MessageField` switches the transport into structured mode. Instead of printing `[data, ...messages]`, it builds one object containing the message, data, and (optionally) timestamp/level fields, and prints just that:

```go
console.New(console.Config{
    MessageField: "msg",
    DateField:    "ts",
    LevelField:   "level",
})

log.Info("structured")
// map[level:info msg:structured ts:2026-04-25T...]
```

Add `Stringify: true` to JSON-encode the object instead of printing it as a Go map:

```go
console.New(console.Config{
    MessageField: "msg",
    Stringify:    true,
})
log.Info("json out")
// {"msg":"json out"}
```

If you only want JSON output, prefer the [structured transport](/transports/structured); it's purpose-built for that.

### Custom Date / Level / Message Functions

```go
console.New(console.Config{
    LevelField: "lvl",
    LevelFn:    func(l loglayer.LogLevel) string { return strings.ToUpper(l.String()) },
    DateField:  "ts",
    DateFn:     func() string { return time.Now().Format(time.RFC822) },
    MessageFn:  func(p loglayer.TransportParams) string { return strings.Join(stringifyMessages(p.Messages), " | ") },
})
```

`MessageFn` receives the full `TransportParams` so you can format based on the level, fields, or anything else.

## Metadata Handling

Map metadata is merged into the same data bag as fields and errors. Struct metadata is JSON-roundtripped into root fields. See [Metadata](/logging-api/metadata) for the design.

## Threat Model: Plaintext, Not for Pipelines

Console's user message string is sanitized for control characters before output (CR, LF, ESC, Unicode bidi controls, zero-width joiners; see `transport.SanitizeMessage`). Field and metadata values, however, are rendered through `fmt`'s `%v` and pass through to the writer in their typed form, including any control characters they happen to contain.

If your service has untrusted input flowing into a logged field and the resulting log lines are read by a viewer or parser that's tricked by control characters, **use [`structured`](/transports/structured) instead**. Structured emits JSON; encoding/json escapes all control characters by default. Console (like pretty) is for the developer's terminal during local dev.
