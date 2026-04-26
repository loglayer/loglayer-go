---
title: log/slog Transport
description: Wrap a *slog.Logger with LogLayer.
---

# log/slog Transport

Wraps a stdlib `*log/slog.Logger`. Map metadata flattens to `slog.Attr`s; struct metadata lands under a configurable key. Per-call `context.Context` attached via `WithCtx` is passed through to `slog.Logger.LogAttrs` so handlers downstream (OpenTelemetry, structured shippers) can extract trace context.

```sh
go get go.loglayer.dev/loglayer/transports/slog
```

## Basic Usage

```go
import (
    "log/slog"
    "os"

    "go.loglayer.dev/loglayer"
    llslog "go.loglayer.dev/loglayer/transports/slog"
)

handler := slog.NewJSONHandler(os.Stderr, nil)
sl := slog.New(handler)

log := loglayer.New(loglayer.Config{
    Transport: llslog.New(llslog.Config{Logger: sl}),
})

log.Info("hello")
// {"time":"...","level":"INFO","msg":"hello"}
```

If you don't pass a `Logger`, the transport constructs one with `slog.NewJSONHandler` writing to `Writer` (default `os.Stderr`).

## Config

```go
type Config struct {
    transport.BaseConfig

    Logger            *slog.Logger // wrap an existing logger
    Writer            io.Writer    // used only when Logger is nil
    MetadataFieldName string       // key for non-map metadata; default "metadata"
}
```

## Fatal Behavior

slog has no fatal level. This transport maps `LogLevelFatal` to `slog.LevelError + 4` so it sorts above Error in any handler that filters by level. The actual `os.Exit(1)` decision is made by the LogLayer core based on `Config.DisableFatalExit`. See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).

## Metadata Handling

### Map metadata → individual `slog.Attr`s

```go
log.WithMetadata(loglayer.Metadata{"requestId": "abc", "n": 42}).Info("served")
// {"level":"INFO","msg":"served","requestId":"abc","n":42}
```

Each map entry becomes a `slog.Any(k, v)` attribute, so slog renders it via the configured handler (JSON, text, or anything custom).

### Struct metadata → nested under `MetadataFieldName`

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")
// {"level":"INFO","msg":"user","metadata":{"id":7,"name":"Alice"}}
```

The JSON handler honors `json:` tags; other handlers may render fields differently.

## context.Context Pass-through

The slog transport is special: it forwards `WithCtx` directly to the underlying `slog.Logger.LogAttrs` call.

```go
import "context"

ctx := context.WithValue(context.Background(), traceKey{}, "trace-abc")
log.WithCtx(ctx).Info("request received")
```

If your slog handler is wired to OpenTelemetry (e.g. via `slogcontext` or a custom handler), the trace context is extracted automatically. See [Go Context](/logging-api/go-context) for the broader pattern.

## Reaching the Underlying Logger

`GetLoggerInstance` returns the underlying `*slog.Logger`:

```go
sl := log.GetLoggerInstance("slog").(*slog.Logger)
sl.With("global", "field").Info("...")
```

## Level Mapping

| LogLayer Level   | slog Level         | Note                                              |
|------------------|--------------------|---------------------------------------------------|
| `LogLevelTrace`  | `LevelDebug`       | slog has no Trace level                           |
| `LogLevelDebug`  | `LevelDebug`       |                                                   |
| `LogLevelInfo`   | `LevelInfo`        |                                                   |
| `LogLevelWarn`   | `LevelWarn`        |                                                   |
| `LogLevelError`  | `LevelError`       |                                                   |
| `LogLevelFatal`  | `LevelError + 4`   | slog has no Fatal; renders as `ERROR+4` in output |
