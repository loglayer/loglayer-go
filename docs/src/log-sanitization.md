---
title: Log Sanitization
description: "Where loglayer-go sanitizes user-controlled strings, what gets stripped, and which transports skip it"
---

# Log Sanitization

Some loglayer-go transports strip control characters from user-controlled strings before writing them. This page documents what's sanitized, where, and why; what isn't sanitized and why that's still safe; and the one developer-issued opt-in that permits authored multi-line content.

If you're writing a custom transport, the [For transport authors](#for-transport-authors) section at the bottom is the part you want.

## Threat model

Three attack classes are in scope:

1. **Log forging.** Untrusted input containing `\n` could write fake follow-up log lines that look like they came from your app. Example: a username `"alice\n2026-01-01 ERROR root: privilege escalation"` printed without sanitization would forge an alert-shaped log line.
2. **Terminal escape smuggling.** Untrusted input containing ANSI ESC (`\x1b`) could inject color codes, move the cursor, clear the screen, or exploit terminal-emulator vulnerabilities. Example: `\x1b]0;evil\x07` rewrites the terminal title; `\x1b[2J\x1b[H` clears the screen.
3. **Trojan Source / hidden content.** Bidi-control characters (U+202E "right-to-left override") visually reorder displayed text without changing the byte sequence; zero-width joiners and zero-width spaces (U+200B–U+200D, U+FEFF) hide content inside what looks like a single token. Example: `"admin‮⁦// safe"` displays as `"admin// safe"` while the real content reads admin+safe-comment-marker.

The defense is `utils/sanitize.Message`, applied at the rendering boundary in transports that target a human terminal. It drops every rune for which `unicode.IsPrint` returns false (with `\t` permitted as the only exception, since terminals interpret tab as column alignment).

## Where sanitization runs

| Site | What it sanitizes | When it fires |
|---|---|---|
| **`cli`** message body | each authored line of the message | every log call |
| **`cli`** user prefix | `WithPrefix(...)` value | when a prefix is set |
| **`cli`** level-prefix overrides | `Config.LevelPrefix` values | once at construction |
| **`cli`** logfmt field values | `fmt.Sprintf("%v", v)` per value | when `ShowFields: true` |
| **`cli`** table cells | `fmt.Sprint(v)` per cell | when metadata is slice-of-map |
| **`pretty`** message body | each authored line of the message | every log call |
| **`console`** message body | each authored line of the message | every log call |
| **`integrations/loghttp`** request fields | `RequestID`, `Method`, `Path`, recovered panic value | every request log emission |

The shared helper for sanitizing message content is [`transport.AssembleMessage(messages []any, sanitize func(string) string) string`](/transports/creating-transports). It applies the sanitize function per element while preserving line boundaries inside `*loglayer.MultilineMessage` values.

## What's stripped

The default `sanitize.Message` drops:

- ANSI ESC (`\x1b`)
- CR (`\r`)
- LF (`\n`): except inside an authored [`loglayer.Multiline`](/logging-api/multiline) value
- C0/C1 control chars (`\x00`–`\x1f`, `\x7f`–`\x9f`)
- Bidi controls (U+202A–U+202E, U+2066–U+2069, etc.)
- Zero-width joiners and spaces (U+200B–U+200D, U+FEFF)
- Other Cf-category format chars

What's preserved:

- All printable Unicode (ASCII letters / digits / punctuation, accented characters, CJK, emoji)
- Tab (`\t`)

## What does NOT get sanitized

`structured` and every wrapper transport (`zerolog`, `zap`, `slog`, `logrus`, `charmlog`, `phuslu`, `sentry`, `otellog`, `gcplogging`, `http`, `datadog`, `testing`) **do not** call `sanitize.Message`. They rely on the JSON encoder downstream to escape control bytes:

```
"\n" → "\\n"
"\x1b" → ""
"‮" → "‮"
```

This is safe for the JSON-shaped sinks because the wire output is meant for log pipelines, log aggregators, and tools like `jq` that interpret JSON-encoded escapes as text. None of them re-emit the raw bytes to a TTY.

::: warning Don't `cat` JSON wrapper-transport output to a TTY without escaping
The `` in JSON is text, but if you pipe wrapper-transport output through a tool that *does* interpret JSON escapes back to bytes (e.g., a homemade pretty-printer that calls `json.Unmarshal` and prints raw strings), you reintroduce the smuggling vector. Use `jq -r .msg` carefully on untrusted log content; prefer `jq` without `-r` (which keeps the JSON escaping) for safety.
:::

Field and metadata values reaching wrapper transports are similarly unsanitized in code; the JSON encoder is the only defense. If your wrapper-transport output flows into a non-JSON sink, audit that path.

## Opting in to authored multi-line content

[`loglayer.Multiline(lines ...any)`](/logging-api/multiline) is the developer's opt-in to permit `\n` *between* authored elements while keeping per-line sanitization for everything else. Each line is still sanitized for ANSI / CR / bidi / ZWSP individually; only the boundaries between elements survive.

```go
// One log call rendered across three lines on cli/pretty/console:
log.Info(loglayer.Multiline(
    "Configuration:",
    "  port: 8080",
    "  host: ::1",
))
```

A bare string with `\n` (no wrapper, no trust) still has the `\n` stripped on those transports. See [Multi-line messages](/logging-api/multiline) for the full contract.

## For transport authors

When you're writing a custom transport, the question is whether to sanitize message content yourself. Use this decision tree:

- **Are you rendering directly to a TTY (or to anything that might be tail-followed in a terminal)?** Sanitize. Use [`transport.AssembleMessage(params.Messages, sanitize.Message)`](/transports/creating-transports) to flatten the message slice with per-line sanitize built in. This handles `*loglayer.MultilineMessage` correctly so authored multi-line content survives.
- **Are you producing JSON for a log pipeline?** Don't sanitize. The encoder handles control bytes. Pre-sanitizing would mangle legitimate log content (a stack trace's `\n` should round-trip).
- **Are you wrapping an existing logger library (zerolog/zap/slog/...)?** Don't sanitize. The underlying library has its own opinions about escaping; reaching past it is invasive and often wrong. The library produces JSON or another structured format that handles escaping itself.

If you're in case 1 (terminal rendering) and your transport sanitizes message bodies, also sanitize anywhere else where user-controlled strings reach the writer:

- The `WithPrefix` value (`params.Prefix`)
- Any `Config.*Prefix` field that accepts user input loaded from environment or config files
- Field and metadata values, if your transport renders them as text (logfmt, table cells, expanded YAML)

The cli transport is the canonical reference for "every text-shaped path is sanitized." Read `transports/cli/cli.go` for the worked pattern.

## Known gaps

- **Multiline-in-fields/metadata.** [`loglayer.Multiline`](/logging-api/multiline) is honored only as a positional message argument. A `*MultilineMessage` value placed inside `WithFields(...)` or `WithMetadata(...)` collapses to one line on cli/pretty/console (terminal sinks sanitize per-value). JSON sinks serialize via `MarshalJSON` to the joined string, so no data is silently lost there.
- **Pretty's expanded YAML mode** doesn't yet honor authored `\n` inside metadata values; it sanitizes per-value like the inline mode. A future change may route through YAML's first-class multi-line scalars; for now, render the multi-line content as a message instead of as a metadata field.

## Reference

- `utils/sanitize.Message(string) string`: the shared sanitizer.
- `transport.AssembleMessage(messages []any, sanitize func(string) string) string`: per-line, Multiline-aware message assembly.
- [Multi-line messages](/logging-api/multiline): the developer-issued opt-in for authored `\n`.
- [Creating Transports](/transports/creating-transports): full transport-authoring guide.
