---
title: Basic Logging
description: The five log level methods and how messages are assembled.
---

# Basic Logging

Once you have a `*loglayer.LogLayer`, the core API is five methods, one per level:

```go
log.Debug("...")
log.Info("...")
log.Warn("...")
log.Error("...")
log.Fatal("...")
```

Each method takes `...any` and joins the arguments with a single space when the entry is rendered:

```go
log.Info("user", 42, "logged in")
// msg: "user 42 logged in"
```

Non-string arguments are formatted with `fmt.Sprintf("%v", arg)`.

## Log Levels

LogLayer defines five numeric levels:

| Constant                  | Value |
|---------------------------|-------|
| `loglayer.LogLevelDebug`  | 10    |
| `loglayer.LogLevelInfo`   | 20    |
| `loglayer.LogLevelWarn`   | 30    |
| `loglayer.LogLevelError`  | 40    |
| `loglayer.LogLevelFatal`  | 50    |

Numeric ordering matters for `SetLevel`. See [Adjusting Log Levels](/logging-api/adjusting-log-levels).

## Fatal Exits the Process

`log.Fatal(...)` dispatches the entry to every transport, then calls `os.Exit(1)`. This matches the Go convention used by `log.Fatal` in the standard library, zerolog, zap, logrus, and others: a fatal log marks the process as unrecoverable.

If you don't want the exit (tests, library code, integration scenarios where the host should decide), set `DisableFatalExit: true` on the config:

```go
log := loglayer.New(loglayer.Config{
    Transport:        structured.New(structured.Config{}),
    DisableFatalExit: true,
})

log.Fatal("logged but no exit") // entry written, process continues
```

`loglayer.NewMock()` enables this automatically. See [Mocking](/logging-api/mocking).

## Prefixes

Set a prefix on the logger so every message starts with it:

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    Prefix:    "[auth]",
})

log.Info("started") // msg: "[auth] started"
```

`WithPrefix("[child]")` returns a child logger with the prefix overridden:

```go
authLog := log.WithPrefix("[auth]")
dbLog   := log.WithPrefix("[db]")

authLog.Info("login ok")  // msg: "[auth] login ok"
dbLog.Info("query took 12ms") // msg: "[db] query took 12ms"
```

The original `log` is not modified.

## Adding Structured Data

To attach data to a single log entry, chain `WithMetadata` and/or `WithError`, then terminate with a level method:

```go
log.WithMetadata(loglayer.Metadata{"userId": 42}).Info("login")
log.WithError(err).Error("failed")
log.WithMetadata(...).WithError(err).Error("...")
```

For data that should appear on **every** log from a logger, use `WithFields`. See [Fields](/logging-api/fields).
