---
title: Multiple Transports
description: Fanning out a single log entry to multiple backends.
---

# Multiple Transports

A LogLayer instance can dispatch every log entry to several transports at once. Set `Transports` (plural) on the config:

```go
log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        console.New(console.Config{
            BaseConfig: transport.BaseConfig{ID: "console"},
        }),
        structured.New(structured.Config{
            BaseConfig: transport.BaseConfig{ID: "ship"},
            Writer:     logFile,
        }),
    },
})

log.Info("user signed in") // both transports receive it
```

## Dispatch Semantics

- Each transport's `SendToLogger` is called sequentially in registration order on the goroutine that called the log method.
- Disabled transports (per `IsEnabled()` or `BaseConfig.Level` filtering) are skipped.
- Transports do not see each other's output.
- The same `TransportParams` value is passed to all of them. Don't mutate it inside a transport, other transports will see your changes.

## Why Sequential, Not Parallel

Most transports are pure formatting + a write. Goroutine setup costs more than the work itself. If a transport does I/O that genuinely blocks, that transport should buffer or queue internally, making the whole dispatch loop async would penalize the common case.

## Per-Transport Level Filtering

Each transport has its own minimum level via `BaseConfig.Level`. This applies *in addition to* the logger's level state, and is the typical way to "log everything to console, ship only warnings":

```go
loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        console.New(console.Config{
            BaseConfig: transport.BaseConfig{ID: "console"},
            // no Level set → defaults to Trace (everything)
        }),
        structured.New(structured.Config{
            BaseConfig: transport.BaseConfig{
                ID:    "ship",
                Level: loglayer.LogLevelWarn,
            },
            Writer: logFile,
        }),
    },
})

log.Info("local-only")  // console only
log.Warn("everywhere")  // both
```

## Adding and Removing at Runtime

```go
log.AddTransport(extraTransport)
log.RemoveTransport("ship")
log.WithFreshTransports(t1, t2) // replace all
```

See [Transport Management](/logging-api/transport-management).

## Single-Transport Optimization

If you're only using one transport, set `Transport` (singular). The core takes a fast path that avoids the loop overhead. This matters in tight loops.

```go
loglayer.New(loglayer.Config{Transport: t}) // fast path
```
