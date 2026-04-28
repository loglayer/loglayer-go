---
title: Zap Transport
description: Wrap a go.uber.org/zap logger with LogLayer.
---

# Zap Transport

Wraps a `*zap.Logger`. Map metadata becomes individual zap fields; struct metadata lands under a configurable key. Fatal-level entries are written via a custom `CheckWriteHook` so the process is **not** terminated, regardless of zap's defaults.

```sh
go get go.loglayer.dev/transports/zap
```

## Basic Usage

```go
import (
    "go.uber.org/zap"

    "go.loglayer.dev"
    llzap "go.loglayer.dev/transports/zap"
)

z, _ := zap.NewProduction()

log := loglayer.New(loglayer.Config{
    Transport: llzap.New(llzap.Config{Logger: z}),
})

log.Info("hello")
// {"level":"info","ts":...,"msg":"hello"}
```

If you don't pass a `Logger`, the transport constructs one with a JSON encoder writing to `Writer` (default `os.Stderr`).

## Config

```go
type Config struct {
    transport.BaseConfig

    Logger            *zap.Logger // wrap an existing logger
    Writer            io.Writer   // used only when Logger is nil
    MetadataFieldName string      // key for non-map metadata; default "metadata"
}
```

## Metadata Handling

### Map metadata → individual fields

```go
log.WithMetadata(loglayer.Metadata{"requestId": "abc", "n": 42}).Info("served")
// {"level":"info","msg":"served","requestId":"abc","n":42}
```

Each map entry becomes a `zap.Any(k, v)` call, so zap renders it however its encoder is configured.

### Struct metadata → nested under `MetadataFieldName`

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")
// {"level":"info","msg":"user","metadata":{"id":7,"name":"Alice"}}
```

zap reflects into the struct via `zap.Any`, which is faster than a JSON roundtrip.

To use a different key per call, wrap in a map:

```go
log.WithMetadata(loglayer.Metadata{"user": User{ID: 7, Name: "Alice"}}).Info("user")
```

Or globally via `MetadataFieldName`:

```go
llzap.New(llzap.Config{
    Logger:            z,
    MetadataFieldName: "payload",
})
```

## Fatal Behavior

zap's `Logger.Fatal` and dispatch via the default fatal hook both call `os.Exit(1)`. **`zap.WithFatalHook(zapcore.WriteThenNoop)` does not work**: zap silently overrides `WriteThenNoop` back to `WriteThenFatal`. To prevent zap from exiting before the core's `DisableFatalExit` check runs, this transport always wraps the supplied logger with a custom no-op hook:

```go
logger := userLogger.WithOptions(zap.WithFatalHook(noopFatalHook{}))
```

The result: zap writes the fatal entry and returns. The core then decides whether to call `os.Exit(1)` based on `Config.DisableFatalExit`. See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).

## Reaching the Underlying Logger

`GetLoggerInstance` returns the (fatal-hook-wrapped) `*zap.Logger`:

```go
z := log.GetLoggerInstance("zap").(*zap.Logger)
z.Sync()
```

This is the wrapped instance, not the original you passed in. For most operations that doesn't matter: fields, sampling, and hooks set before passing the logger to LogLayer are preserved.

## Level Mapping

| LogLayer Level   | zap Level       | Note                        |
|------------------|-----------------|-----------------------------|
| `LogLevelDebug`  | `DebugLevel`    |                             |
| `LogLevelInfo`   | `InfoLevel`     |                             |
| `LogLevelWarn`   | `WarnLevel`     |                             |
| `LogLevelError`  | `ErrorLevel`    |                             |
| `LogLevelFatal`  | `FatalLevel`    | written but no `os.Exit`    |
