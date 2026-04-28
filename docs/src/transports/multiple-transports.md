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

- Transports are called sequentially in registration order on the goroutine that called the log method.
- Disabled transports (per the transport's enabled flag or its `BaseConfig.Level` filter) are skipped.
- Transports do not see each other's output.

Authoring a custom transport? See [Creating Transports](/transports/creating-transports) for the dispatch-side contract (immutable params, error handling, concurrency).

## Why Sequential, Not Parallel

Most transports are pure formatting + a write. Goroutine setup costs more than the work itself. If a transport does I/O that genuinely blocks, that transport should buffer or queue internally, making the whole dispatch loop async would penalize the common case.

## Per-Transport Level Filtering

Each transport has its own minimum level via `BaseConfig.Level`. This applies *in addition to* the logger's level state, and is the typical way to "log everything to console, ship only warnings":

```go
loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        console.New(console.Config{
            BaseConfig: transport.BaseConfig{ID: "console"},
            // no Level set → defaults to Debug (everything)
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
log.SetTransports(t1, t2) // replace all
```

See [Transport Management](/transports/management).

## Single vs Multiple

For one transport, set `Transport` (singular). For more than one, use `Transports` (a slice). Both produce a working logger; pick the one that matches your call site.

```go
loglayer.New(loglayer.Config{Transport: t})                                // single
loglayer.New(loglayer.Config{Transports: []loglayer.Transport{t1, t2}})   // multiple
```

## Recipe: pretty in dev, structured to a file, ship to Datadog

A realistic production setup. Pretty is colorized terminal output for the developer attached to the process; structured writes JSON-per-line to a rolling file for local correlation; Datadog ships everything to the Logs HTTP intake. Each transport has its own minimum level so the noisy `Debug` lines stay local.

```go
import (
    "os"

    "go.loglayer.dev"
    "go.loglayer.dev/transport"
    "go.loglayer.dev/transports/datadog"
    "go.loglayer.dev/transports/pretty"
    "go.loglayer.dev/transports/structured"
)

logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
if err != nil {
    panic(err)
}

log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        pretty.New(pretty.Config{
            BaseConfig: transport.BaseConfig{ID: "pretty", Level: loglayer.LogLevelDebug},
        }),
        structured.New(structured.Config{
            BaseConfig: transport.BaseConfig{ID: "file", Level: loglayer.LogLevelInfo},
            Writer:     logFile,
        }),
        datadog.New(datadog.Config{
            BaseConfig: transport.BaseConfig{ID: "datadog", Level: loglayer.LogLevelWarn},
            APIKey:     os.Getenv("DATADOG_API_KEY"),
            Site:       datadog.SiteUS1,
            Service:    "checkout-api",
        }),
    },
})
```

Every emission fans out to all three transports in registration order. Each transport's own minimum level filters independently: `log.Debug(...)` reaches pretty only; `log.Info(...)` reaches pretty and the file; `log.Warn(...)` and above reach all three.

For routing rules beyond level filters (e.g. send `audit.*` only to the file, never to Datadog), see [Groups](/logging-api/groups).
