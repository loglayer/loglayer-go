---
title: Sampling Plugin
description: "Drop a fraction of log emissions to keep volume and cost under control."
---

# Sampling Plugin

<ModuleBadges path="plugins/sampling" />

`plugins/sampling` keeps log volume bounded by dropping a fraction of emissions before any transport sees them. Three strategies built on the [`SendGate`](/plugins/creating-plugins#sendgate) hook:

- **`FixedRate(rate)`**: independent random draw per emission. Best for "I want about 1% of debug logs."
- **`FixedRatePerLevel(map)`**: same shape, per-level rates. Levels not in the map pass unconditionally.
- **`Burst(n, window)`**: keep the first N emissions per rolling window, drop the rest. Best for hard caps like "no more than 100 logs/second."

```sh
go get go.loglayer.dev/plugins/sampling
```

Pure Go, no dependencies (uses `math/rand/v2` from the stdlib).

## Basic Usage

```go
import (
    "go.loglayer.dev"
    "go.loglayer.dev/plugins/sampling"
    "go.loglayer.dev/transports/structured"
)

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})

// Keep 1% of all emissions.
log.AddPlugin(sampling.FixedRate(0.01))
```

## `FixedRate(rate float64)`

Independent Bernoulli draw per emission. `rate >= 1` keeps everything (no-op gate), `rate <= 0` drops everything.

```go
log.AddPlugin(sampling.FixedRate(0.1)) // about 10% kept
```

Random selection uses [`math/rand/v2`](https://pkg.go.dev/math/rand/v2), which is goroutine-safe and seeded by the runtime. For deterministic sampling in tests, register your own `SendGate` consulting a controlled source.

## `FixedRatePerLevel(rates map[LogLevel]float64)`

Per-level rate. Levels absent from the map pass unconditionally, so the typical shape is "rate-limit only the noisy levels":

```go
log.AddPlugin(sampling.FixedRatePerLevel(map[loglayer.LogLevel]float64{
    loglayer.LogLevelTrace: 0.01, // 1% of trace
    loglayer.LogLevelDebug: 0.1,  // 10% of debug
    // info / warn / error / fatal / panic: kept (not in map)
}))
```

The map is snapshot at construction time; mutating it afterwards does not affect the live sampler.

## `Burst(n int, window time.Duration)`

Hard rate cap. Keeps the first `n` emissions per rolling window of `window`, drops the rest until the window resets.

```go
log.AddPlugin(sampling.Burst(100, time.Second)) // at most 100 logs/sec
```

`n <= 0` drops everything. `window <= 0` keeps everything (no time limit). The window is shared across levels and transports; for distinct caps, register multiple `Burst` plugins (note that the default plugin ID `"sampling-burst"` would collide; pass distinct IDs via your own `SendGate` if needed).

## Composition

Multiple sampling plugins compose: an emission is kept only when every gate returns true. Combining `FixedRate` and `Burst` gives "1% of logs, capped at 100/sec":

```go
log.AddPlugin(sampling.FixedRate(0.01))
log.AddPlugin(sampling.Burst(100, time.Second))
```

The `FixedRate` gate runs first and drops 99% of traffic. The `Burst` gate then caps the 1% that made it through.

## Caveats

- Sampling decisions happen **per transport** (the `SendGate` hook is called once per `(entry, transport)` pair). With multiple transports, `FixedRate` and `FixedRatePerLevel` perform an independent Bernoulli draw for each transport — one transport may keep the entry while another drops it. `Burst` shares a single counter across transports, so each per-transport call consumes a slot from the same window (e.g. with two transports and a 100/sec cap, you get ~50 fully-delivered entries per second). If you want per-transport sampling rates, use [Group routing](/logging-api/groups) with different transports per group.
- The `Burst` sampler holds an internal mutex. Lock scope is tiny (a counter check + time comparison) but under extreme contention it can serialize the dispatch path. For most workloads this is negligible compared to the emission cost itself.
- Sampling does **not** affect plugin hooks that ran earlier (`OnFieldsCalled`, `OnMetadataCalled`). It only gates dispatch to transports.
