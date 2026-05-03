---
title: Multi-line messages with loglayer.Multiline
description: "Author multi-line message content that survives the cli/pretty/console sanitizer"
---

# Multi-line messages with `loglayer.Multiline`

`loglayer.Multiline(lines ...any)` lets you author a message that renders on multiple lines through `cli`, `pretty`, and `console`. It's a developer-issued token of trust: the wrapper signals that the line boundaries between elements were authored by you, so the sanitizer in those transports preserves the `\n` between them while still stripping ANSI / control bytes inside each line.

## Quickstart

```go
import "go.loglayer.dev/v2"

log.Info(loglayer.Multiline(
    "Configuration:",
    "  port: 8080",
    "  host: ::1",
))
// Configuration:
//   port: 8080
//   host: ::1
```

`Multiline` accepts any number of arguments and treats each one as a separate authored line. Non-string arguments are formatted with `fmt.Sprintf("%v", v)` (Stringer is honored). Strings containing embedded `\n` are split at construction, so `Multiline("a\nb")` and `Multiline("a", "b")` are interchangeable.

## Why bare `\n` doesn't work

If you write `log.Info("Header:\n  port: 8080")` without the wrapper, the cli, pretty, and console transports collapse it to one line:

```
Header:  port: 8080
```

The sanitizer at those rendering boundaries strips `\n` from message strings to defeat two attacks:

1. **Log forging:** untrusted input containing `\n` could write fake follow-up log lines that look like they came from your app.
2. **Terminal escape smuggling:** untrusted input containing ANSI ESC, bidi overrides (Trojan Source), or zero-width joiners could inject color codes, hide content, or exploit terminal vulns.

`Multiline` opts you out of the line-collapsing rule for this *one* call, while keeping every other defense intact. Each authored line is still individually sanitized.

## What's preserved, what's stripped

| Inside one authored line | Across authored lines |
|---|---|
| ANSI ESC: stripped | `\n` boundary: preserved |
| CR: stripped | |
| Bidi overrides (U+202E etc.): stripped | |
| Zero-width joiners / spaces: stripped | |

A bare string with `\n` (no wrapper, no trust) still has the `\n` stripped. `Multiline("\x1b", "[31mred")` cannot reconstruct an ANSI escape across the boundary because each line is sanitized in isolation before joining.

## Per-transport behavior

| Transport | `Multiline("a","b")` | `"Header:", Multiline("a","b")` |
|---|---|---|
| **cli** | `a\nb` (level-colored, on the level's writer) | `Header: a\nb` |
| **pretty** | `[ts] [INFO] a\nb` | `[ts] [INFO] Header: a\nb` |
| **console** | `{"msg":"a\nb",...}` (MessageField mode) or `a\nb [k=v ...]` (default) | analogous |
| **structured** | `{"msg":"a\nb",...}` | `{"msg":"Header: a\nb",...}` |
| **zerolog / zap / slog / logrus / charmlog / phuslu** | underlying logger writes `"a\nb"` | underlying logger writes `"Header: a\nb"` |
| **sentry / otellog / gcplogging / http / datadog / testing** | same: `Stringer` fallback joins with `"\n"` | same |

## With a prefix

`WithPrefix` folds the prefix into the first authored line; subsequent lines are unchanged.

```go
log.WithPrefix("[svc]").Info(loglayer.Multiline("a", "b"))
// [svc] a
// b
```

## Inside fields or metadata

::: warning Not honored inside Fields or Metadata
`Multiline` only applies when it appears as a positional message argument (`log.Info(loglayer.Multiline(...))`, `log.Error(loglayer.Multiline(...))`, etc.). Inside `WithFields(...)` or `WithMetadata(...)`, terminal transports (cli, pretty, console) still sanitize each value to a single line, so a `Multiline` value placed there gets collapsed.

JSON sinks (structured + every wrapper transport) serialize `Multiline` values via `MarshalJSON` to the `\n`-joined string, so no data is silently lost in those sinks.

If you need multi-line value rendering for fields specifically, file an issue describing the use case; the right shape is a separate design (probably routing through pretty's expanded-YAML mode).
:::

Plugin authors who walk `params.Messages` should preserve `*MultilineMessage` values; see [Creating plugins](/plugins/creating-plugins#preserving-multilinemessage-values). One built-in plugin where the wrapper is intentionally collapsed is `fmtlog`'s format-string mode; see [Format Strings](/plugins/fmtlog#interaction-with-multiline) for the workaround.

For the full picture of what gets sanitized where (across every transport), see [Log Sanitization](/log-sanitization).
