---
title: charmbracelet/log Transport
description: Wrap a github.com/charmbracelet/log logger with LogLayer.
---

# charmbracelet/log Transport

<ModuleBadges path="transports/charmlog" />

Wraps an existing `*charmbracelet/log.Logger`. Map metadata flattens to alternating key/value pairs; struct metadata lands under a configurable key.

```sh
go get go.loglayer.dev/transports/charmlog
```

The package name is `charmlog` (since both the import path's last element and the package itself are `log`, which would collide with the stdlib).

## Basic Usage

```go
import (
    "os"

    clog "github.com/charmbracelet/log"

    "go.loglayer.dev"
    llcharm "go.loglayer.dev/transports/charmlog"
)

cl := clog.NewWithOptions(os.Stderr, clog.Options{
    Level:           clog.InfoLevel,
    ReportTimestamp: true,
})

log := loglayer.New(loglayer.Config{
    Transport: llcharm.New(llcharm.Config{Logger: cl}),
})

log.Info("hello")
// 2026-04-25 12:00:00 INFO hello
```

If you don't pass a `Logger`, the transport constructs one writing to `Writer` (default `os.Stderr`).

## Config

```go
type Config struct {
    transport.BaseConfig

    Logger            *charmbracelet/log.Logger // wrap an existing logger
    Writer            io.Writer                 // used only when Logger is nil
    MetadataFieldName string                    // key for non-map metadata; default "metadata"
}
```

## Fatal Behavior

charmbracelet's `Logger.Fatal()` calls `os.Exit(1)`, but `Logger.Log(FatalLevel, msg, keyvals...)` does not. This wrapper always dispatches via `Log(level, ...)`, so charmbracelet writes the fatal entry and returns. The core then decides whether `os.Exit(1)` is called after dispatch. See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).

## Metadata Handling

### Map metadata → individual key/value pairs

```go
log.WithMetadata(loglayer.Metadata{"requestId": "xyz", "n": 42}).Info("served")
// INFO served requestId=xyz n=42
```

Each map entry becomes a `(key, value)` pair in the variadic `keyvals` argument to `Log`.

### Struct metadata → nested under `MetadataFieldName`

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")
// INFO user metadata={ID:7 Name:Alice}
```

charmbracelet renders the struct via its default formatter. Exact output shape depends on whether you've configured `JSONFormatter`, `TextFormatter`, or the default colored output.

To use a different key per call, wrap in a map:

```go
log.WithMetadata(loglayer.Metadata{"user": User{ID: 7, Name: "Alice"}}).Info("user")
```

Or globally via `MetadataFieldName`:

```go
llcharm.New(llcharm.Config{
    Logger:            cl,
    MetadataFieldName: "payload",
})
```

## Reaching the Underlying Logger

`GetLoggerInstance` returns the wrapped `*charmbracelet/log.Logger`:

```go
cl := log.GetLoggerInstance("charmlog").(*clog.Logger)
cl.SetLevel(clog.DebugLevel)
```

## Level Mapping

| LogLayer Level   | charmbracelet Level |
|------------------|---------------------|
| `LogLevelDebug`  | `DebugLevel`        |
| `LogLevelInfo`   | `InfoLevel`         |
| `LogLevelWarn`   | `WarnLevel`         |
| `LogLevelError`  | `ErrorLevel`        |
| `LogLevelFatal`  | `FatalLevel`        |
