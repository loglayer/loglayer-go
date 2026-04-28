---
title: logrus Transport
description: Wrap a github.com/sirupsen/logrus logger with LogLayer.
---

# logrus Transport

<ModuleBadges path="transports/logrus" />

Wraps an existing `*logrus.Logger`. Map metadata flattens via `Entry.WithFields`; struct metadata lands under a configurable key.

```sh
go get go.loglayer.dev/transports/logrus
```

## Basic Usage

```go
import (
    "os"

    "github.com/sirupsen/logrus"

    "go.loglayer.dev"
    lllogrus "go.loglayer.dev/transports/logrus"
)

l := logrus.New()
l.Out = os.Stderr
l.Formatter = &logrus.JSONFormatter{}

log := loglayer.New(loglayer.Config{
    Transport: lllogrus.New(lllogrus.Config{Logger: l}),
})

log.Info("hello")
// {"level":"info","msg":"hello","time":"..."}
```

If you don't pass a `Logger`, the transport constructs one writing to `Writer` (default `os.Stderr`) at TraceLevel.

## Config

```go
type Config struct {
    transport.BaseConfig

    Logger            *logrus.Logger // wrap an existing logger
    Writer            io.Writer      // used only when Logger is nil
    MetadataFieldName string         // key for non-map metadata; default "metadata"
}
```

## Fatal Behavior

`logrus.Logger.Fatal` and `Logger.Log(FatalLevel, ...)` call the logger's `ExitFunc`, which by default is `os.Exit(1)`. To prevent logrus from exiting before the core's [`DisableFatalExit`](/configuration#disablefatalexit) check can run, this transport **always builds a fresh `*logrus.Logger` that copies the supplied logger's settings** (`Out`, `Hooks`, `Formatter`, `ReportCaller`, `Level`, `BufferPool`) but with `ExitFunc` set to a no-op.

Two consequences:

- The user's original `*logrus.Logger` is never mutated; its `ExitFunc` is preserved.
- The wrapped copy returned by `GetLoggerInstance` is the no-op-ExitFunc version, not the original. For most operations this doesn't matter; configurations set on the original (formatter, hooks, etc.) are all preserved.

The core then decides whether `os.Exit(1)` is called after dispatch. See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).

## Metadata Handling

### Map metadata → individual fields

```go
log.WithMetadata(loglayer.Metadata{"requestId": "xyz", "n": 42}).Info("served")
// {"level":"info","msg":"served","n":42,"requestId":"xyz","time":"..."}
```

Each map entry becomes a key in `logrus.Fields` passed to `WithFields`.

### Struct metadata → nested under `MetadataFieldName`

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")
// {"level":"info","metadata":{"id":7,"name":"Alice"},"msg":"user",...}
```

To use a different key per call, wrap in a map:

```go
log.WithMetadata(loglayer.Metadata{"user": User{ID: 7, Name: "Alice"}}).Info("user")
```

Or globally via `MetadataFieldName`:

```go
lllogrus.New(lllogrus.Config{
    Logger:            l,
    MetadataFieldName: "payload",
})
```

## Reaching the Underlying Logger

`GetLoggerInstance` returns the wrapped (no-op-ExitFunc) `*logrus.Logger`:

```go
l := log.GetLoggerInstance("logrus").(*logrus.Logger)
l.AddHook(myHook)
```

## Level Mapping

| LogLayer Level   | logrus Level   |
|------------------|----------------|
| `LogLevelDebug`  | `DebugLevel`   |
| `LogLevelInfo`   | `InfoLevel`    |
| `LogLevelWarn`   | `WarnLevel`    |
| `LogLevelError`  | `ErrorLevel`   |
| `LogLevelFatal`  | `FatalLevel`   |
