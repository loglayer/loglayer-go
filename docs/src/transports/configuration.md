---
title: Transport Configuration
description: Wire transports into a LogLayer at construction time and pick the right BaseConfig values.
---

# Transport Configuration

Transports are wired into a `*loglayer.LogLayer` at construction time via the `Config` struct. This page covers the construction-time options. For runtime mutation (add, remove, replace, reach the underlying logger), see [Transport Management](/transports/management).

## Single transport

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})
```

`Config.Transport` is the typical entry point for a logger that emits to one place.

## Multiple transports

```go
log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        pretty.New(pretty.Config{}),
        structured.New(structured.Config{Writer: f}),
    },
})
```

Use `Config.Transports` (plural) to fan out to several transports at once. Setting both `Transport` and `Transports` panics with `loglayer.ErrTransportAndTransports`. See [Multiple Transports](/transports/multiple-transports) for fan-out semantics and ordering.

## BaseConfig

Every built-in transport embeds `transport.BaseConfig`, which carries three fields shared across the line-up:

| Field | Type | Default | Purpose |
|-------|------|---------|---------|
| `ID` | `string` | auto-generated | Stable handle for runtime management calls (`RemoveTransport(id)`, `GetLoggerInstance(id)`). |
| `Disabled` | `bool` | `false` | Suppress this transport's emissions without removing it. Equivalent to calling `SetEnabled(false)` after construction. |
| `Level` | `loglayer.LogLevel` | `LogLevelTrace` | Per-transport minimum level. Defaults to accepting every level; the logger's own level state is the primary filter. Set this when you want a transport to receive only entries at or above a specific level (e.g. an error-only sink in a fan-out). |

```go
console.New(console.Config{
    BaseConfig: transport.BaseConfig{
        ID:    "console",
        Level: loglayer.LogLevelInfo,
    },
})
```

## Transport IDs

Every transport has an `ID()` method; set the ID at construction via `transport.BaseConfig{ID: ...}`. Every transport's `New` returns a `*Transport` wrapping its `BaseTransport`, so when the ID is empty the constructed value carries the auto-generated ID (`auto-transport-<hex>`) and you can read it back from `ID()` before wiring the transport into the logger. If you later call `RemoveTransport(id)` or `GetLoggerInstance(id)` with the wrong string, the call silently under-delivers: `RemoveTransport` returns `false`, `GetLoggerInstance` returns `nil`, and the transport stays in the dispatch list. Confirm the assigned ID with `tr.ID()` before relying on it. Keep the constructed transport handle: `*LogLayer` offers no ID-lookup method, so once the transport is handed to the config, its ID is unrecoverable from the logger.

The base config lives in the shared `transport` package. Import it alongside the transport:

```go
import "go.loglayer.dev/v3/transport"

console.New(console.Config{
    BaseConfig: transport.BaseConfig{ID: "console"},
})
```

An ID is only needed when the logger will manage that transport at runtime: `RemoveTransport(id)`, `GetLoggerInstance(id)`, and replace-by-ID (`AddTransport` replaces an existing transport with the same ID instead of duplicating it). For transports you set up once and never touch (a single console renderer, a one-shot test transport), leaving `ID` empty is fine: the auto-generated ID still works for routing and group dispatch, you just won't have a stable handle for management calls.

The random-ID hazard applies most in fan-out setups: `RemoveTransport("ship")` removes nothing (and returns `false`) when the "ship" transport was constructed without an ID and got `auto-transport-...`. See [Multiple Transports → Transport IDs](/transports/multiple-transports#transport-ids) for a worked example.

::: warning Auto-generated IDs are random
Leaving `BaseConfig.ID` empty assigns a random ID per construction. Never call `RemoveTransport` or `GetLoggerInstance` with an ID you copied from an earlier run, and do not key routing config off an auto-generated ID: it changes every process start. Always set explicit IDs for any transport you intend to manage by ID.
:::

## Enabling and disabling per environment

When you wire several transports into one logger, the same code typically runs in dev, CI, and production but you don't want every transport active in every environment. Common patterns:

- **Local dev**: pretty terminal, no shipping. You don't want CI runs or laptop runs hitting the production Datadog account.
- **CI**: structured JSON to stdout (so the test runner captures it), no shipping.
- **Staging / production**: structured to a file plus the network shipper (Datadog, Loki, OTel).

Two ways to wire that:

### Build the slice from environment

The most common pattern: include or exclude each transport at construction time based on env. Cheap, explicit, no runtime mutation needed.

```go
transports := []loglayer.Transport{
    pretty.New(pretty.Config{
        BaseConfig: transport.BaseConfig{ID: "pretty", Level: loglayer.LogLevelDebug},
    }),
}
if os.Getenv("APP_ENV") == "production" {
    transports = append(transports, datadog.New(datadog.Config{
        BaseConfig: transport.BaseConfig{ID: "datadog", Level: loglayer.LogLevelInfo},
        APIKey:     os.Getenv("DATADOG_API_KEY"),
    }))
}
log := loglayer.New(loglayer.Config{Transports: transports})
```

### Construct everything, disable per env via `BaseConfig.Disabled`

If your code reads cleaner with a fixed transport list, set `Disabled: true` on the ones that shouldn't run in this environment. The transport is still constructed (so its config is validated) but `SendToLogger` is a no-op.

```go
isProd := os.Getenv("APP_ENV") == "production"

log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        pretty.New(pretty.Config{
            BaseConfig: transport.BaseConfig{ID: "pretty", Disabled: isProd},
        }),
        datadog.New(datadog.Config{
            BaseConfig: transport.BaseConfig{ID: "datadog", Disabled: !isProd},
            APIKey:     os.Getenv("DATADOG_API_KEY"),
        }),
    },
})
```

### Toggling at runtime

Cast the transport's exported `SetEnabled(bool)` method (inherited from `BaseTransport`) to flip without rebuilding. Useful for admin endpoints or feature-flag-driven rollouts:

```go
if t, ok := tr.(interface{ SetEnabled(bool) }); ok {
    t.SetEnabled(false)
}
```

For routing rules beyond on/off (e.g. only ship `audit.*` entries to Datadog), see [Groups](/logging-api/groups).

## See Also

- [Transport Management](/transports/management), runtime mutation of the transport list.
- [Multiple Transports](/transports/multiple-transports), fan-out semantics and dispatch order.
- [Creating Transports](/transports/creating-transports), implementing the `Transport` interface yourself.
