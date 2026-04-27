---
title: Migrating from zerolog
description: "Replace zerolog calls with LogLayer's fluent API while keeping your zerolog logger."
---

# Migrating from `rs/zerolog`

zerolog's chained-builder API and `Event.Msg("...")` terminator are familiar Go territory. LogLayer's [`transports/zerolog`](/transports/zerolog) wrapper hands assembled entries straight to a `*zerolog.Logger` you already configured, so the on-the-wire output and writer stay the same. What changes is the call shape — and what you gain on top.

## Why migrate

zerolog is fast and ergonomic. Adopt LogLayer if you want:

- A single API across your service even when some downstream libraries log via `slog` or `zap`.
- A path to swap zerolog for another logger later without touching application code.
- Plugins (redact, OTel trace injection, custom hooks) that work uniformly across transports.
- Built-in fan-out: ship to zerolog *and* a pretty terminal output *and* an HTTP backend simultaneously.

If none of those apply, stay with zerolog.

## Setup

Before:

```go
import (
    "os"
    zlog "github.com/rs/zerolog"
)

logger := zlog.New(os.Stderr).With().Timestamp().Logger()
```

After:

```go
import (
    "os"
    "go.loglayer.dev"
    zerologtransport "go.loglayer.dev/transports/zerolog"
    zlog "github.com/rs/zerolog"
)

zl := zlog.New(os.Stderr).With().Timestamp().Logger()

log := loglayer.New(loglayer.Config{
    Transport: zerologtransport.New(zerologtransport.Config{Logger: &zl}),
})
```

Same `*zerolog.Logger`, same writer, same JSON shape on the wire. `log.GetLoggerInstance("zerolog").(*zerolog.Logger)` returns the underlying logger if a library wants it.

## API translation

| zerolog                                                          | LogLayer                                                              |
|------------------------------------------------------------------|-----------------------------------------------------------------------|
| `logger.Info().Msg("served")`                                    | `log.Info("served")`                                                  |
| `logger.Info().Str("requestId", rid).Msg("served")`              | `log.WithFields(loglayer.Fields{"requestId": rid}).Info("served")`    |
| `logger.Info().Int("n", 42).Msg("event")`                        | `log.WithMetadata(loglayer.Metadata{"n": 42}).Info("event")`          |
| `logger.With().Str("svc", "api").Logger()` (sub-logger)          | `log.WithFields(loglayer.Fields{"svc": "api"})`                       |
| `logger.Err(err).Msg("failed")`                                  | `log.WithError(err).Error("failed")`                                  |
| `logger.Info().Interface("user", user).Msg("hi")`                | `log.WithMetadata(user).Info("hi")` (struct nests under `MetadataFieldName`) |
| `zerolog.Ctx(ctx)` (logger from context)                         | `loglayer.FromContext(ctx)` — see [Go Context](/logging-api/go-context) |

## What changes for your zerolog logger

Nothing. The zerolog wrapper:

- Maps map-shaped metadata via `Event.Fields(map)` (flattened at root).
- Nests struct/scalar metadata via `Event.Interface(MetadataFieldName, val)`.
- Routes `LogLevelFatal` through `WithLevel(zerolog.FatalLevel)` instead of `Fatal()`, so your zerolog logger never calls `os.Exit` directly. The LogLayer core handles the exit decision via `Config.DisableFatalExit`.
- Preserves `LogLevelTrace` (zerolog has it natively).

## What you gain

- **Type-safe Fields vs Metadata distinction.** `WithFields` is for persistent state; `WithMetadata` is for per-call payload. The compiler catches accidental mixing.
- **Multi-transport fan-out.** Add a pretty terminal transport during development without touching application code.
- **Plugins**: redact secrets before they reach zerolog, inject trace IDs, etc.
- **Per-transport routing** via Groups.

## What stays the same

- The zerolog logger and its writer (file, stdout, stderr, custom `io.Writer`).
- Your sub-logger configuration (`.With().Timestamp().Caller().Logger()`).
- zerolog's level filter — entries below the underlying logger's level still don't render.
- All the wire-format conventions you've configured (timestamp format, field names, etc.).
