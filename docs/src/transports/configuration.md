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
| `Level` | `loglayer.LogLevel` | `LogLevelTrace` | Per-transport minimum level. Stacks on top of the logger's own level state. |

```go
console.New(console.Config{
    BaseConfig: transport.BaseConfig{
        ID:    "console",
        Level: loglayer.LogLevelInfo,
    },
})
```

## Transport IDs

`BaseConfig.ID` is optional. When you omit it, the transport gets an auto-generated ID, so multiple no-ID transports never collide. **Supply your own ID** when you'll later need to address that specific transport: `RemoveTransport(id)`, `GetLoggerInstance(id)`, and `AddTransport`'s replace-by-ID semantics all key off the string you set.

```go
console.New(console.Config{
    BaseConfig: transport.BaseConfig{ID: "console"},
})
```

For transports you set up once and never touch (a single console renderer, a one-shot test transport), leaving `ID` empty is fine: the auto-generated ID still works for routing, you just won't have a stable handle for management calls.

## See Also

- [Transport Management](/transports/management), runtime mutation of the transport list.
- [Multiple Transports](/transports/multiple-transports), fan-out semantics and dispatch order.
- [Creating Transports](/transports/creating-transports), implementing the `Transport` interface yourself.
