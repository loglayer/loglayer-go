---
title: Format Strings
description: "Opt-in fmt.Sprintf semantics for multi-arg log calls."
---

# Format Strings

<ModuleBadges path="plugins/fmtlog" />

`fmtlog` is a one-line plugin that opts a logger into `fmt.Sprintf`-style format strings. After registration, every call where the first message is a string and there are extra arguments is rewritten via `fmt.Sprintf(messages[0], messages[1:]...)` before downstream `MessageHook`s run.

```sh
go get go.loglayer.dev/plugins/fmtlog/v2
```

`fmtlog` is its own Go module under `go.loglayer.dev/plugins/fmtlog/v2`, with no third-party dependencies beyond the main `go.loglayer.dev/v2` module it implements `Plugin` against.

## Basic Usage

```go
import (
    "go.loglayer.dev/v2"
    "go.loglayer.dev/plugins/fmtlog/v2"
    "go.loglayer.dev/transports/structured/v2"
)

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})
log.AddPlugin(fmtlog.New())

log.Info("user %d signed in", 1234)
// → msg: "user 1234 signed in"

log.Error("request %s failed: %v", reqID, err)
// → msg: "request abc-123 failed: connection refused"
```

The plugin composes naturally with the builder chain:

```go
log.WithMetadata(loglayer.Metadata{"reqId": reqID}).
    WithError(err).
    Error("request %s failed", reqID)
```

## Why a Plugin Instead of `Infof` Methods

LogLayer deliberately doesn't ship `Infof` / `Warnf` / `Errorf` / etc. on `*LogLayer`. Two reasons:

1. **Structured-first.** The message field is a label, not a sentence. Format strings encourage burying values inside the message that would be more queryable as `WithMetadata` keys. The core API stays out of the way.
2. **Opt-in.** Some teams use `log.Info("got %d users", n)` *intending* a literal message ("`%d`" is in the text on purpose). Adding format-string semantics globally would surprise them. Registering `fmtlog.New()` is an explicit "I want printf semantics."

The plugin form mirrors how [TypeScript loglayer](https://loglayer.dev) handles the same need.

## Behavior Summary

| Call shape | Without `fmtlog` | With `fmtlog` |
|---|---|---|
| `log.Info("plain")` | `"plain"` | `"plain"` |
| `log.Info("100% complete")` | `"100% complete"` | `"100% complete"` |
| `log.Info("user %d", 42)` | `"user %d 42"` (space-joined) | `"user 42"` |
| `log.Info(42, "extra")` | `"42 extra"` (joined) | `"42 extra"` (first arg isn't a string; bypassed) |

The plugin's preconditions are:

- `len(messages) >= 2`
- `messages[0]` is a `string`

If either fails, the messages slice passes through untouched.

## Performance

`fmtlog.New()` is a single `MessageHook`. Per-call cost when the plugin doesn't fire (single-arg call, or first arg isn't a string): one type assertion and a length check. When it does fire: one `fmt.Sprintf` call.

The `Sprintf` only runs when the entry actually dispatches. Filtered calls (level off, plugin gate, etc.) never reach the hook, so no formatting cost is paid:

```go
log.SetLevel(loglayer.LogLevelWarn)
log.Debug("expensive %v", computeStuff()) // computeStuff() runs (Go semantics);
                                          // Sprintf is skipped.
```

For deferred argument evaluation when the *args themselves* are expensive, gate on level explicitly:

```go
if log.IsLevelEnabled(loglayer.LogLevelDebug) {
    log.Debug("expensive %v", computeStuff())
}
```

## Plugin Hook

`fmtlog.New()` registers a single `MessageHook` named `"fmtlog"`. Other plugins that read or rewrite `Messages` see the resolved string when their hook runs after `fmtlog`. To control ordering relative to other `MessageHook`s, register them in the desired order; hooks run in registration order.
