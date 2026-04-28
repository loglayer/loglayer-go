---
title: slog Handler
description: "slog.Handler that routes every slog.Info(...) call through your loglayer pipeline."
---

# slog Handler

<ModuleBadges path="integrations/sloghandler" bundled />

`integrations/sloghandler` is a `log/slog.Handler` backed by a loglayer logger. Once installed, every `slog.Info(...)` call (yours or your dependencies') flows through loglayer's plugin pipeline, multi-transport fan-out, group routing, and level filtering.

```sh
go get go.loglayer.dev/integrations/sloghandler
```

This is the **slog → loglayer** direction. If you want the opposite (use loglayer's API and emit through a `*slog.Logger` you've already configured), see the [slog Transport](/transports/slog).

## When to Use This

- Your dependencies log via `*slog.Logger` (a growing convention in the ecosystem) and you want their output redacted, traced, fanned out, or routed by group, just like your own.
- You're standardizing on `slog.Default()` across an application but still want loglayer's pipeline behind it.
- You have call sites already written against slog and don't want to rewrite them.

## Basic Usage

```go
import (
    "log/slog"

    "go.loglayer.dev"
    "go.loglayer.dev/integrations/sloghandler"
    "go.loglayer.dev/plugins/redact"
    "go.loglayer.dev/transports/structured"
)

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})
log.AddPlugin(redact.New(redact.Config{Keys: []string{"password"}}))

slog.SetDefault(slog.New(sloghandler.New(log)))

slog.Info("user signed in", "userId", 42, "password", "hunter2")
// {"level":"info","time":"...","msg":"user signed in","userId":42,"password":"[REDACTED]"}
```

The redact plugin runs even though the call site is `slog.Info(...)`. Same for `oteltrace`, `datadogtrace`, fan-out across multiple transports, group routing, and runtime level mutation.

## Mapping

### Levels

| slog                             | loglayer                  |
|----------------------------------|---------------------------|
| `slog.LevelDebug` and below      | `LogLevelDebug`           |
| `slog.LevelInfo`                 | `LogLevelInfo`            |
| `slog.LevelWarn`                 | `LogLevelWarn`            |
| `slog.LevelError` and above      | `LogLevelError`           |

slog has no Fatal level, so values at or above `slog.LevelError` pin to `LogLevelError`. **A slog emission cannot trigger loglayer's `os.Exit(1)`.** If you need Fatal, call `log.Fatal(...)` directly on the loglayer side.

### Attrs

| slog API                            | loglayer effect                                  |
|-------------------------------------|--------------------------------------------------|
| `slog.With(...)` / `WithAttrs`      | Persistent fields on the derived logger          |
| Inline attrs in `slog.Info("m", ...)` | Per-call fields on that one emission           |
| `Handler.WithGroup("g")`            | Subsequent attrs nest under `g` in the output    |
| `slog.Group("g", ...)` value        | Members nested under `g`                         |
| `slog.Group("", ...)` value         | Members inlined at the parent level (slog spec)  |
| Empty group (no attrs ever added)   | Dropped from output (slog spec)                  |
| Attr with empty key                 | Dropped (slog spec)                              |
| `slog.LogValuer`                    | Resolved before encoding                         |

### Native types preserved

`slog.Int64` produces an `int64` field, `slog.Time` produces a `time.Time`, etc. Transports that special-case these types (e.g. structured emitting RFC3339 for `time.Time`) work the same as they do when called via loglayer's own API.

### Context

The `context.Context` passed to `slog.InfoContext(ctx, ...)` (and the handler's `Handle` method generally) is forwarded to dispatch-time plugin hooks via `TransportParams.Ctx`. Plugins like [`oteltrace`](/plugins/oteltrace) and [`datadogtrace`](/plugins/datadogtrace) extract trace IDs from this context, so `slog.InfoContext(ctx, ...)` participates in distributed tracing the same way `log.WithCtx(ctx).Info(...)` does on the loglayer side.

## Underlying Loglayer Fields

Persistent fields set on the loglayer logger (via `log.WithFields(...)` before installing the handler) are preserved on every emission that comes through the handler:

```go
log := loglayer.New(loglayer.Config{Transport: ...}).
    WithFields(loglayer.Fields{"service": "api"})
slog.SetDefault(slog.New(sloghandler.New(log)))

slog.Info("hi")
// {"...","msg":"hi","service":"api"}

slog.Info("with-attr", "k", "v")
// {"...","msg":"with-attr","service":"api","k":"v"}
```

## Mixing slog and Loglayer Call Sites

You can keep using loglayer directly for your own code and let slog handle the dependency-emitted logs:

```go
log := loglayer.New(loglayer.Config{Transport: structured.New(structured.Config{})})
slog.SetDefault(slog.New(sloghandler.New(log)))

// Your code
log.WithMetadata(loglayer.Metadata{"userId": 42}).Info("served")

// Some library you import
slog.Info("ratelimited", "client", "abc")
```

Both paths run through the same plugin pipeline and the same transports.

## Error Attrs

`slog.Any("err", err)` arrives as a field with the original `error` value. The transport decides how to serialize it (default is whatever the configured `ErrorSerializer` does, otherwise the JSON encoder calls `Error()`).

If you want loglayer's structured error treatment (`{"err": {"message": ...}}` via the configured `ErrorSerializer`), call `log.WithError(err).Info(...)` directly on the loglayer side rather than passing the error as a slog attr.

## Concurrency

The handler is safe under concurrent emission. `WithAttrs` and `WithGroup` return new handler values without mutating the receiver, so derived `*slog.Logger`s shared across goroutines do not race.

## Source / Caller Info

`slog.New` always captures `slog.Record.PC` for every emission. The handler forwards that PC into loglayer as a `*Source` via `RawLogEntry.Source`, so a structured transport renders it under `SourceFieldName` (default `"source"`) automatically:

```go
slog.SetDefault(slog.New(sloghandler.New(log)))
slog.Info("hello")
// {"level":"info","time":"...","msg":"hello","source":{"function":"main.main","file":"/app/main.go","line":12}}
```

No need to set `Config.Source.Enabled` on the loglayer side; the slog frontend already paid the capture cost. (If you call loglayer's own `log.Info(...)` directly and want the same source rendering, see [Configuration → Source](/configuration#source-caller-info).)

## Differences from Other slog Handlers

- **Levels above slog.Error don't escalate to Fatal.** Other handlers don't have Fatal at all; this one suppresses it deliberately so a custom slog level can't accidentally exit the process.
- **No HandlerOptions on this side.** Filtering by level is done on the loglayer side (`log.SetLevel`, per-group levels, `Config.Level`). The handler's `Enabled` consults the loglayer logger.

## Related

- [slog Transport](/transports/slog): the opposite direction. Loglayer emits through a `*slog.Logger` backend.
- [Plugins](/plugins/): every built-in plugin works the same when called via slog.
- [Multi-Transport](/transports/multiple-transports): slog emissions fan out to every configured transport.
