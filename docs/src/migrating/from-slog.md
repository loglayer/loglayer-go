---
title: Migrating from log/slog
description: "Replace stdlib slog calls with LogLayer's fluent API while keeping your existing handlers."
---

# Migrating from `log/slog`

You don't have to throw away your slog handler stack to adopt LogLayer. The [`transports/slog`](/transports/slog) wrapper hands assembled entries to a `*slog.Logger` you already configured, so JSON handlers, OTel-enabled handlers, custom routing — all keep working. What you change is the *call* surface: from slog's vararg key-value pattern to LogLayer's fluent builder.

## Why migrate

slog is a fine baseline, but the API has friction:

- `slog.With(key, val, key, val, ...)` puts you one missed argument away from a misaligned log entry.
- Per-call structured payloads (request bodies, error objects) need `slog.Group` or `slog.Any` ceremony.
- Switching loggers later — adding pretty terminal output for dev, shipping to Datadog — means rewriting every call site.

LogLayer separates "persistent fields" from "per-call metadata" at the type level (`Fields` vs `Metadata`), accepts arbitrary values for metadata (no `slog.Any` wrapping), and lets you swap the underlying transport without touching application code.

## Setup

Before:

```go
import "log/slog"

handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
sl := slog.New(handler)
slog.SetDefault(sl)
```

After:

```go
import (
    "log/slog"
    "go.loglayer.dev"
    llslog "go.loglayer.dev/transports/slog"
)

handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
sl := slog.New(handler)

log := loglayer.New(loglayer.Config{
    Transport: llslog.New(llslog.Config{Logger: sl}),
})
```

Same handler, same JSON output. The `*slog.Logger` is intact and reachable via `log.GetLoggerInstance("slog").(*slog.Logger)` if some library you don't control wants the underlying logger.

## API translation

| slog call                                                | LogLayer equivalent                                                   |
|----------------------------------------------------------|-----------------------------------------------------------------------|
| `sl.Info("served")`                                      | `log.Info("served")`                                                  |
| `sl.Info("served", "n", 42)`                             | `log.WithMetadata(loglayer.Metadata{"n": 42}).Info("served")`         |
| `sl.With("requestId", rid).Info("served")`               | `log.WithFields(loglayer.Fields{"requestId": rid}).Info("served")`    |
| `sl.WithGroup("auth").Info("login")`                     | LogLayer groups are different — see [Groups](/logging-api/groups). For nested attrs, use a struct/map metadata. |
| `sl.Error("failed", "err", err)`                         | `log.WithError(err).Error("failed")`                                  |
| `sl.LogAttrs(ctx, slog.LevelInfo, "served", attrs...)`   | `log.WithCtx(ctx).Info("served")` (no attrs ceremony — pass via Fields/Metadata) |

## What changes for your handlers

Nothing. The slog wrapper:

- Forwards `WithCtx(ctx)` to `slog.Logger.LogAttrs(ctx, ...)` so your handlers receive the same `context.Context` they always have. Handlers reading trace IDs out of context (e.g. an OTel-bridged handler) keep working.
- Maps map-shaped metadata to individual `slog.Attr`s (flattened at root).
- Nests struct/scalar metadata under `MetadataFieldName` (default `"metadata"`).
- Maps `LogLevelTrace` to `slog.LevelDebug` (slog has no Trace level).
- Maps `LogLevelFatal` to `slog.LevelError + 4` so it sorts above Error in custom handlers.

## What you gain

- **Pretty terminal output for dev** by adding the [`pretty`](/transports/pretty) transport in development. Fan out to slog *and* pretty simultaneously via `Config.Transports`.
- **Per-call metadata that takes any value**, not just key-value pairs. `log.WithMetadata(myStruct).Info(...)` works.
- **Plugins**: `redact`, `oteltrace`, custom hooks at lifecycle points. See [Plugins](/plugins/).
- **Group routing**: send error logs to one transport, debug to another. See [Groups](/logging-api/groups).

## What stays the same

- Your handler stack (JSON, text, OTel, custom).
- The Go context plumbing — `WithCtx` reaches `slog.Handler.Handle(ctx, ...)`.
- Your level filtering (slog handler's `Level` option still applies).
- `slog.Default()` still works for libraries you don't own.
