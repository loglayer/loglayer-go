---
title: Transport Management
description: Add, remove, and replace transports at runtime; reach the underlying logger instance.
---

# Transport Management

A `*loglayer.LogLayer` holds a list of transports. After construction, you can mutate the list at runtime: add a new shipper, remove a noisy console transport, hot-swap the entire set. All mutators are safe to call from any goroutine, including concurrently with emission.

For construction-time setup (wiring transports via `Config.Transport` / `Config.Transports`, picking IDs and levels), see [Transport Configuration](/transports/configuration).

## Add

`AddTransport(transports...)` appends. If a transport with the same `ID` already exists it is **replaced**, not duplicated:

```go
log.AddTransport(structured.New(structured.Config{
    BaseConfig: transport.BaseConfig{ID: "ship"},
    Writer:     logFile,
}))
```

This is the easiest way to enable shipping in production while keeping your dev console intact.

## Remove

`RemoveTransport(id)` deletes the transport with that ID and returns whether one was found:

```go
removed := log.RemoveTransport("ship")
if !removed {
    // no-op: nothing was registered with that ID
}
```

## Replace

`SetTransports(transports...)` replaces the entire list. Existing transports are dropped:

```go
log.SetTransports(
    console.New(console.Config{BaseConfig: transport.BaseConfig{ID: "console"}}),
    structured.New(structured.Config{BaseConfig: transport.BaseConfig{ID: "ship"}, Writer: f}),
)
```

## Reach the Underlying Logger

Some transports wrap a third-party logger. `GetLoggerInstance(id)` returns that wrapped value, so you can call backend-specific methods when LogLayer's API doesn't cover what you need:

```go
log := loglayer.New(loglayer.Config{
    Transport: llzero.New(llzero.Config{
        BaseConfig: transport.BaseConfig{ID: "zerolog"},
        Logger:     &z,
    }),
})

if zerologLogger, ok := log.GetLoggerInstance("zerolog").(*zlog.Logger); ok {
    zerologLogger.Hook(...) // use zerolog-specific feature
}
```

For transports without an underlying library (console, structured), `GetLoggerInstance` returns `nil`. The `testing` transport returns its `*TestLoggingLibrary`.

## Concurrency

`AddTransport`, `RemoveTransport`, and `SetTransports` publish a new immutable transport set via `atomic.Pointer`, so the dispatch hot path only loads a pointer. Concurrent mutators on the same logger serialize via an internal mutex.
