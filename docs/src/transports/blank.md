---
title: Blank Transport
description: A bring-your-own-function Transport for prototyping or one-off integrations.
---

# Blank Transport

<ModuleBadges path="transports/blank" bundled />

The `blank` transport delegates `SendToLogger` to a function you supply inline. Use it for:

- **Prototyping** a new transport without creating a full package.
- **One-off integrations** (forward to a metrics system, push to a queue, post to an HTTP endpoint) that don't justify their own transport package.
- **Tests** that want to inspect raw `TransportParams` without setting up the [testing transport](/transports/testing).

If you find yourself repeating the same `blank.Config` across the codebase, promote it to its own transport package using the [Creating Transports](/transports/creating-transports) template.

```sh
go get go.loglayer.dev/transports/blank
```

## Basic Usage

```go
import (
    "go.loglayer.dev"
    "go.loglayer.dev/transport"
    "go.loglayer.dev/transports/blank"
)

log := loglayer.New(loglayer.Config{
    Transport: blank.New(blank.Config{
        BaseConfig: transport.BaseConfig{ID: "metrics"},
        ShipToLogger: func(p loglayer.TransportParams) {
            if p.LogLevel >= loglayer.LogLevelError {
                metricsClient.Increment("app.errors", 1)
            }
        },
    }),
})

log.Error("payment failed")
```

## Config

```go
type Config struct {
    transport.BaseConfig

    // ShipToLogger is invoked for every entry that passes the level filter.
    // If nil, entries are silently discarded.
    ShipToLogger func(params loglayer.TransportParams)
}
```

## Behavior

- The transport's `BaseConfig.Level` filter still applies. Set `Level: loglayer.LogLevelError` to receive only error+ entries.
- If `ShipToLogger` is nil, the transport silently drops everything that passes the filter. This makes a no-op blank transport useful as a placeholder during development.
- `GetLoggerInstance` returns nil; there is no underlying library.
- The wrapper itself is safe under concurrent emission. If your `ShipToLogger` mutates shared state, you are responsible for synchronizing it.

## Patterns

### Forward to multiple sinks

Pair with [multi-transport fan-out](/transports/multiple-transports) so the blank transport sees every entry alongside your real transports:

```go
loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        structured.New(structured.Config{ /* JSON to stdout */ }),
        blank.New(blank.Config{
            BaseConfig: transport.BaseConfig{ID: "audit"},
            ShipToLogger: func(p loglayer.TransportParams) {
                if p.LogLevel >= loglayer.LogLevelWarn {
                    auditChan <- formatAudit(p)
                }
            },
        }),
    },
})
```

### Inspect entries in a test

When you don't need the typed `LogLine` capture of the [testing transport](/transports/testing), this is shorter:

```go
var captured []loglayer.TransportParams
log := loglayer.New(loglayer.Config{
    Transport: blank.New(blank.Config{
        BaseConfig: transport.BaseConfig{ID: "spy"},
        ShipToLogger: func(p loglayer.TransportParams) {
            captured = append(captured, p)
        },
    }),
    DisableFatalExit: true,
})
```

## When to Promote

Once you've used the same `blank.Config` in two places, it's time to write a real transport. The [Creating Transports](/transports/creating-transports) page shows the minimal template.
