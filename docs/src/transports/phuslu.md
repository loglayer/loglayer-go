---
title: phuslu/log Transport
description: Wrap a github.com/phuslu/log logger with LogLayer.
---

# phuslu/log Transport

<ModuleBadges path="transports/phuslu" />

Wraps an existing `*phuslu/log.Logger`. Map metadata flattens to fields via `Entry.Any(k, v)`; struct metadata lands under a configurable key.

```sh
go get go.loglayer.dev/transports/phuslu/v2
go get github.com/phuslu/log
```

## Basic Usage

```go
import (
    "os"

    plog "github.com/phuslu/log"

    "go.loglayer.dev/v2"
    llphuslu "go.loglayer.dev/transports/phuslu/v2"
)

p := &plog.Logger{
    Level:  plog.InfoLevel,
    Writer: &plog.IOWriter{Writer: os.Stderr},
}

log := loglayer.New(loglayer.Config{
    Transport: llphuslu.New(llphuslu.Config{Logger: p}),
})

log.Info("hello")
// {"time":"...","level":"info","message":"hello"}
```

If you don't pass a `Logger`, the transport constructs one writing to `Writer` (default `os.Stderr`).

## Config

```go
type Config struct {
    transport.BaseConfig

    Logger *phuslu/log.Logger // wrap an existing logger
    Writer io.Writer          // used only when Logger is nil
}
```

## Fatal Behavior

::: danger phuslu always exits on Fatal
phuslu calls `os.Exit(1)` from every fatal-level dispatch path, including `Logger.WithLevel(FatalLevel).Msg(...)`. **This wrapper cannot suppress that behavior.** A `log.Fatal(...)` through the phuslu transport WILL terminate the process even when [`Config.DisableFatalExit`](/configuration#disablefatalexit) is set to `true`.

If you need fatal paths to not exit (tests, library code, integration scenarios), use a different transport for those scenarios. The [structured](/transports/structured), [zerolog](/transports/zerolog), and [zap](/transports/zap) transports all honor `DisableFatalExit`.
:::

For non-fatal levels, the wrapper dispatches via `Logger.WithLevel(level).Msg(...)`, preserving phuslu's full pipeline (formatters, hooks, async writers).

## Metadata Handling

<!--@include: ./_partials/metadata-field-name.md-->

### Map metadata → individual fields

```go
log.WithMetadata(loglayer.Metadata{"requestId": "xyz", "n": 42}).Info("served")
// {"time":"...","level":"info","message":"served","requestId":"xyz","n":42}
```

Each map entry becomes an `Entry.Any(k, v)` call.

### Struct metadata nests under the metadata key

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")
// {"time":"...","level":"info","message":"user","metadata":{"id":7,"name":"Alice"}}
```

To use a different key per call, wrap in a map:

```go
log.WithMetadata(loglayer.Metadata{"user": User{ID: 7, Name: "Alice"}}).Info("user")
```

Or globally via the core's `MetadataFieldName` (which also nests map metadata under the same key):

```go
loglayer.New(loglayer.Config{
    Transport:         llphuslu.New(llphuslu.Config{Logger: p}),
    MetadataFieldName: "payload",
})
```

## Reaching the Underlying Logger

`GetLoggerInstance` returns the wrapped `*phuslu/log.Logger`:

```go
p := log.GetLoggerInstance("phuslu").(*plog.Logger)
p.SetLevel(plog.DebugLevel)
```

## Level Mapping

| LogLayer Level   | phuslu Level   |
|------------------|----------------|
| `LogLevelTrace`  | `TraceLevel`   |
| `LogLevelDebug`  | `DebugLevel`   |
| `LogLevelInfo`   | `InfoLevel`    |
| `LogLevelWarn`   | `WarnLevel`    |
| `LogLevelError`  | `ErrorLevel`   |
| `LogLevelFatal`  | `FatalLevel`   |
| `LogLevelPanic`  | `PanicLevel`   |
