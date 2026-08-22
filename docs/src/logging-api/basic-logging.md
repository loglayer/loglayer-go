---
title: Basic Logging
description: The seven log level methods and how messages are assembled.
---

# Basic Logging

Once you have a `*loglayer.LogLayer`, the core API is seven methods, one per level:

```go
log.Trace("...")
log.Debug("...")
log.Info("...")
log.Warn("...")
log.Error("...")
log.Fatal("...")
log.Panic("...")
```

Each method takes `...any` and joins the arguments with a single space when the entry is rendered:

```go
log.Info("user", 42, "logged in")
// msg: "user 42 logged in"
```

Non-string arguments are formatted with `fmt.Sprintf("%v", arg)`.

## Log Levels

LogLayer defines seven numeric levels:

| Constant                  | Value | Method      | Notes                              |
|---------------------------|-------|-------------|------------------------------------|
| `loglayer.LogLevelTrace`  | 5     | `Trace(...)` | Below Debug; very fine-grained diagnostic |
| `loglayer.LogLevelDebug`  | 10    | `Debug(...)` |                                    |
| `loglayer.LogLevelInfo`   | 20    | `Info(...)`  |                                    |
| `loglayer.LogLevelWarn`   | 30    | `Warn(...)`  |                                    |
| `loglayer.LogLevelError`  | 40    | `Error(...)` |                                    |
| `loglayer.LogLevelFatal`  | 50    | `Fatal(...)` | Calls `os.Exit(1)` after dispatch  |
| `loglayer.LogLevelPanic`  | 60    | `Panic(...)` | Calls `panic(msg)` after dispatch  |

Numeric ordering matters for `SetLevel`. See [Adjusting Log Levels](/logging-api/adjusting-log-levels).

## Fatal Exits the Process

`log.Fatal(...)` dispatches the entry to every transport, then calls `os.Exit(1)`. This matches the Go convention used by `log.Fatal` in the standard library, zerolog, zap, logrus, and others: a fatal log marks the process as unrecoverable. Because the exit runs **after** dispatch, every transport still sees the fatal entry; see each transport's Fatal Behavior section for wrapper-specific caveats. When a `Fatal` entry is dispatched, any transport that implements `io.Closer` is closed along the way (capped by `Config.TransportCloseTimeout`), so async transports (HTTP, Datadog) flush pending entries before the exit.

If you don't want the exit (tests, library code, integration scenarios where the host should decide), set `DisableFatalExit: true` on the config:

```go
log := loglayer.New(loglayer.Config{
    Transport:        structured.New(structured.Config{}),
    DisableFatalExit: true,
})

log.Fatal("logged but no exit") // entry written, process continues
```

`loglayer.NewMock()` enables this automatically. See [Mocking](/logging-api/mocking).

::: warning Fatal is not for long-running services
`os.Exit` does not run `defer`s: deferred cleanup (auto-updater re-exec, graceful shutdown, connection drains) is skipped, and the exit code is always `1`. Config changes or log writes in flight are just lost. For long-running services, prefer `log.Error(...)` at the failure point plus an `os.Exit(1)` at a single coordinator that has drained first. Set `DisableFatalExit: true` at the root if call sites can't be trusted to use `Error`.

The default Fatal path does flush `io.Closer` transports before exiting (capped by `Config.TransportCloseTimeout`). If you set `DisableFatalExit: true` and call `os.Exit` yourself, that flush no longer runs: close `io.Closer` transports (HTTP, Datadog) explicitly before your own exit to guarantee delivery.
:::

## Panic Panics the Goroutine

`log.Panic(...)` dispatches the entry, then calls `panic(<joined message>)`. Unlike Fatal, the panic is recoverable: a `defer recover()` higher up the call stack can catch it and continue. Use Panic when you want a logged unrecoverable error that a caller (or framework, like `chi.Recoverer`) can intercept.

```go
defer func() {
    if r := recover(); r != nil {
        log.WithMetadata(loglayer.M{"panic": r}).Error("recovered")
    }
}()

log.Panic("invariant violated") // entry written, then panics with "invariant violated"
```

There is no `DisablePanicExit` knob: Panic always panics, matching zerolog / zap / logrus convention. To suppress in tests, recover in the calling goroutine. Async transports are NOT pre-flushed (closing them would break callers that recover and keep emitting); only Fatal pays that cost.

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

All examples on this page construct the logger with `loglayer.New`, which panics when the config is invalid (no transport). When the config comes from a runtime source (env vars, a config file, a secrets manager), use `loglayer.Build` instead; it returns `(*LogLayer, error)` with the same validation. See [New vs Build](/configuration#new-vs-build).

## stdlib `log` and `io.Writer` Bridges

Third-party libraries often accept a `*log.Logger` or an `io.Writer` and emit one line per call. Two adapter methods on `*LogLayer` turn each line into a loglayer emission so you can plug those libraries straight into your pipeline:

- `log.Writer(level) io.Writer`
- `log.NewLogLogger(level) *log.Logger` (mirrors `slog.NewLogLogger`)

Drop the result into anything that takes the corresponding type:

```go
import (
    "net/http"

    "go.loglayer.dev/v3"
)

srv := &http.Server{
    Addr:     ":8080",
    Handler:  mux,
    ErrorLog: log.NewLogLogger(loglayer.LogLevelError),
}
```

Or for a plain `io.Writer`:

```go
w := log.Writer(loglayer.LogLevelInfo)
fmt.Fprintln(w, "from a third-party library")
```

Each line becomes one entry through the full pipeline (plugins, fan-out, group routing, level state). Trailing newlines are stripped so loglayer's own delimiters aren't doubled. Cost is one full dispatch per line, so for high-volume sources (a busy HTTP server's error log under attack) pair with the [sampling plugin](/plugins/sampling) to cap volume.
