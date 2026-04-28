---
title: Adjusting Log Levels
description: Enable, disable, and threshold log levels at runtime.
---

# Adjusting Log Levels

LogLayer maintains per-level enable/disable state plus a master on/off switch. Both are checked before a log entry reaches any transport. All level mutators are safe to call from any goroutine, so live runtime toggling (SIGUSR1, an admin endpoint flipping debug on, etc.) works without coordination.

## Setting a Threshold

`SetLevel(level)` enables the given level and everything above it; disables everything below.

```go
log.SetLevel(loglayer.LogLevelWarn)

log.Debug("dropped")   // ignored
log.Info("dropped")    // ignored
log.Warn("kept")       // emitted
log.Error("kept")      // emitted
```

Level ordering is `Debug < Info < Warn < Error < Fatal`.

## Enabling / Disabling Individual Levels

If you want a non-contiguous set (for example, `Info` and `Error` but not `Warn`), toggle each level directly:

```go
log.DisableLevel(loglayer.LogLevelWarn)
log.EnableLevel(loglayer.LogLevelDebug)
```

These do not change other levels' state.

## Master Switch

`DisableLogging()` suppresses every level regardless of per-level state. `EnableLogging()` restores the previous per-level configuration.

```go
log.DisableLogging()
log.Info("never emitted")
log.Error("never emitted")

log.EnableLogging()
log.Info("emitted again")
```

You can also disable from construction:

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    Disabled:  true,
})
```

## Inspecting Level State

```go
if log.IsLevelEnabled(loglayer.LogLevelDebug) {
    expensive := buildDebugPayload()
    log.WithMetadata(expensive).Debug("snapshot")
}
```

This is useful when assembling the metadata is itself expensive.

## Transport-Level Filtering

Each transport also has a `Level` field on its `BaseConfig`. A transport will skip entries below its own minimum, regardless of the logger's level state. This lets you have one transport recording at `Debug` for local debugging and another at `Warn` for shipping:

```go
loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        console.New(console.Config{
            BaseConfig: transport.BaseConfig{ID: "console"}, // defaults to Trace (accepts every level)
        }),
        structured.New(structured.Config{
            BaseConfig: transport.BaseConfig{
                ID:    "ship",
                Level: loglayer.LogLevelWarn, // only warn+ get shipped
            },
            Writer: logFile,
        }),
    },
})
```

## Child Loggers Inherit Level State

`Child()` clones the level configuration. Mutations on the child do not propagate to the parent.

```go
log.SetLevel(loglayer.LogLevelInfo)
child := log.Child()

child.SetLevel(loglayer.LogLevelDebug)
log.Debug("dropped, parent still at info")
child.Debug("kept, child at debug")
```

See [Child Loggers](/logging-api/child-loggers).
