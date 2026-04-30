---
title: Zerolog Transport
description: Wrap a github.com/rs/zerolog logger with LogLayer.
---

# Zerolog Transport

<ModuleBadges path="transports/zerolog" />

Wraps an existing `*zerolog.Logger`. Map metadata merges as fields; struct metadata lands under a configurable key. Fatal-level entries are written via `WithLevel` so the process is **not** terminated, regardless of zerolog's defaults.

```sh
go get go.loglayer.dev/transports/zerolog
go get github.com/rs/zerolog
```

## Basic Usage

```go
import (
    zlog "github.com/rs/zerolog"
    "os"

    "go.loglayer.dev"
    llzero "go.loglayer.dev/transports/zerolog"
)

z := zlog.New(os.Stderr).With().Timestamp().Logger()

log := loglayer.New(loglayer.Config{
    Transport: llzero.New(llzero.Config{Logger: &z}),
})

log.Info("hello")
// {"level":"info","time":"...","message":"hello"}
```

If you don't pass a `Logger`, the transport constructs one writing to `Writer` (default `os.Stderr`).

## Config

```go
type Config struct {
    transport.BaseConfig

    Logger *zerolog.Logger // wrap an existing logger
    Writer io.Writer       // used only when Logger is nil
}
```

## Metadata Handling

<!--@include: ./_partials/metadata-field-name.md-->

### Map metadata → fields at root

```go
log.WithMetadata(loglayer.Metadata{"requestId": "abc", "n": 42}).Info("served")
// {"level":"info","time":"...","message":"served","requestId":"abc","n":42}
```

### Struct metadata nests under the metadata key

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")
// {"level":"info","time":"...","message":"user","metadata":{"id":7,"name":"Alice"}}
```

Zerolog's `Interface` field handler reflects directly into the struct, so the value is encoded once at write time without an extra JSON roundtrip.

Use a different key per call by wrapping in a map:

```go
log.WithMetadata(loglayer.Metadata{"user": User{ID: 7, Name: "Alice"}}).Info("user")
```

Or globally via the core's `MetadataFieldName` (which also nests map metadata under the same key):

```go
loglayer.New(loglayer.Config{
    Transport:         llzero.New(llzero.Config{Logger: &z}),
    MetadataFieldName: "payload",
})
```

## Fields

Map fields are merged at the root via zerolog's `Fields`:

```go
log.WithFields(loglayer.Fields{"service": "api"})
log.Info("request")
// {"level":"info","message":"request","service":"api",...}
```

If `FieldsKey` is set on the LogLayer config, the fields are nested first by the core, then merged at root by zerolog. The result appears as a single nested object:

```go
loglayer.New(loglayer.Config{
    Transport: llzero.New(llzero.Config{Logger: &z}),
    FieldsKey: "fields",
})

log.WithFields(loglayer.Fields{"requestId": "abc"})
log.Info("hi")
// {"level":"info","message":"hi","fields":{"requestId":"abc"}}
```

## Fatal Behavior

The wrapper routes fatal entries through `logger.WithLevel(zerolog.FatalLevel)` rather than `.Fatal()`, so it does not trigger zerolog's built-in `os.Exit`. The core's `DisableFatalExit` then decides whether `os.Exit(1)` is called after dispatch. See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).

```go
// Default: fatal exits via core
log.Fatal("unrecoverable")

// Opt-out
log = loglayer.New(loglayer.Config{
    Transport:        llzero.New(llzero.Config{Logger: &z}),
    DisableFatalExit: true,
})
log.Fatal("logged but no exit")
```

## Reaching the Underlying Logger

`GetLoggerInstance` returns the wrapped `*zerolog.Logger`:

```go
z := log.GetLoggerInstance("zerolog").(*zlog.Logger)
z.Hook(myHook)
```

## Level Mapping

| LogLayer Level   | zerolog Level    |
|------------------|------------------|
| `LogLevelTrace`  | `TraceLevel`     |
| `LogLevelDebug`  | `DebugLevel`     |
| `LogLevelInfo`   | `InfoLevel`      |
| `LogLevelWarn`   | `WarnLevel`      |
| `LogLevelError`  | `ErrorLevel`     |
| `LogLevelFatal`  | `FatalLevel`     |
| `LogLevelPanic`  | `PanicLevel`     |
