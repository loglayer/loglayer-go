---
title: Transport Management
description: Add, remove, and replace transports at runtime; reach the underlying logger instance.
---

# Transport Management

A `*loglayer.LogLayer` holds a list of transports. You configure them at construction time via `Config.Transport` or `Config.Transports`, and you can mutate the list at runtime. Mutators are safe to call from any goroutine, including concurrently with emission.

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

`WithFreshTransports(transports...)` replaces the entire list. Existing transports are dropped:

```go
log.WithFreshTransports(
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

## Why IDs Matter

IDs are how you address a specific transport later. If you only have one transport and never plan to swap it, you can leave `ID` empty, but `GetLoggerInstance("")` will look it up by empty-string key, and `RemoveTransport("")` won't help if you ever add a second.

A safe default: name every transport.

```go
console.New(console.Config{
    BaseConfig: transport.BaseConfig{ID: "console"},
})
```

## See Also

- [Multiple Transports](/transports/multiple-transports), fan-out semantics and dispatch order.
- [Creating Transports](/transports/creating-transports), implementing the `Transport` interface yourself.
