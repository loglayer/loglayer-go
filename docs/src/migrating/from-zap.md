---
title: Migrating from zap
description: "Replace zap calls with LogLayer's fluent API while keeping your *zap.Logger."
---

# Migrating from `uber-go/zap`

zap's strongly-typed field constructors (`zap.String`, `zap.Int`, etc.) trade ergonomics for performance. LogLayer's [`transports/zap`](/transports/zap) wrapper hands assembled entries to a `*zap.Logger` you already configured, so the writer, encoders, and sampling all keep working. What changes is the call shape.

## Why migrate

zap is fast. Adopt LogLayer if you want:

- The same API surface across services, even ones using `slog` or `zerolog` underneath.
- A path to swap zap for another logger later without touching call sites.
- Per-call metadata that takes any value, not field-by-field constructors.
- Plugins, group routing, and multi-transport fan-out.

If you're squeezing every nanosecond and don't need any of the above, stay with zap.

## Setup

Before:

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()
```

After:

```go
import (
    "go.loglayer.dev"
    zaptransport "go.loglayer.dev/transports/zap"
    "go.uber.org/zap"
)

zl, _ := zap.NewProduction()
defer zl.Sync()

log := loglayer.New(loglayer.Config{
    Transport: zaptransport.New(zaptransport.Config{Logger: zl}),
})
```

Same `*zap.Logger`, same encoder, same sink. `log.GetLoggerInstance("zap").(*zap.Logger)` returns the underlying logger.

## API translation

| zap                                                              | LogLayer                                                              |
|------------------------------------------------------------------|-----------------------------------------------------------------------|
| `logger.Info("served")`                                          | `log.Info("served")`                                                  |
| `logger.Info("served", zap.Int("n", 42))`                        | `log.WithMetadata(loglayer.Metadata{"n": 42}).Info("served")`         |
| `logger.With(zap.String("svc", "api"))` (sub-logger)             | `log.WithFields(loglayer.Fields{"svc": "api"})`                       |
| `logger.Error("failed", zap.Error(err))`                         | `log.WithError(err).Error("failed")`                                  |
| `logger.Info("hi", zap.Object("user", userObj))`                 | `log.WithMetadata(userObj).Info("hi")` (struct nests under `MetadataFieldName`) |
| `logger.Sugar()` for `zap.SugaredLogger`                         | LogLayer's API is already vararg-friendly; no sugar layer needed.     |

## What changes for your zap logger

Nothing functionally. The zap wrapper:

- Builds a `[]zap.Field` from map-shaped metadata via `zap.Any(k, v)` per entry.
- Nests struct/scalar metadata via `zap.Any(MetadataFieldName, val)`.
- Wraps your `*zap.Logger` with `zap.WithFatalHook(noopFatalHook{})` so fatal entries don't call `os.Exit` from zap; the LogLayer core decides via `Config.DisableFatalExit`. Without this, zap silently re-enables `WriteThenFatal` even if you opt out, so the wrapper installs the hook unconditionally.
- Maps `LogLevelTrace` to `zap.DebugLevel` (zap has no Trace).

## What you gain

- **Type-safe Fields vs Metadata distinction** at the LogLayer level — no more "is this a `zap.Field` or just a `key, val` pair" confusion.
- **Per-call structured payloads** without the per-type field constructor (`zap.String`, `zap.Int`, ...).
- **Multi-transport** — pretty for dev *plus* zap for production *plus* HTTP shipping.
- **Plugins**: redact secrets, inject trace IDs, custom hooks at lifecycle points.

## What stays the same

- The `*zap.Logger` and its encoder/sink configuration (production preset, custom config, sampling).
- zap's level filter (`AtomicLevel`) — entries below the level the zap logger is configured at still don't render.
- `defer zl.Sync()` semantics — your shutdown path is unchanged.

## A note on performance

The wrapper builds a `[]zap.Field` per emission via `zap.Any` for each metadata key, which is slower than zap's native typed constructors. Most services won't notice, but in nanosecond-sensitive code paths consider whether the LogLayer wins (multi-transport, plugins, per-call metadata) outweigh the per-call overhead. The repo's `bench_test.go` quantifies the difference; run `go test -bench=. -benchmem -run=^$ .` from the repo root for current numbers.
