---
title: Benchmarks
description: Per-call cost of LogLayer for Go, broken down by adoption choice.
---

# Benchmarks

LogLayer is a thin layer over your transport. This page measures its per-call cost so you can see what each adoption choice buys you.

::: tip How these were measured
`go test -bench=. -benchmem -run=^$` against a no-op writer, Go 1.25. Host CPU is an AMD EPYC 7413 (24 cores); the benchmark runs inside a VM with 16 cores assigned. Each benchmark exercises the entire dispatch path: emission method → plugin pipeline (zero plugins where unstated) → transport → discard writer.

Absolute numbers will vary on your hardware; the **deltas between rows** are what's portable. Use these to pick between abstraction strategies, not to predict end-to-end throughput.
:::

## Picking a setup

Four primary adoption paths, in order of overhead on a simple message:

| Setup | Cost | When to choose |
|---|---:|---|
| Wrap zerolog | 142 ns / 1 alloc | Already on zerolog, or want maximum throughput. |
| Standalone `structured` | 353 ns / 3 allocs | No third-party logger dep; recommended for new projects. |
| Wrap zap | 470 ns / 1 alloc | Already on zap, or need its `zapcore` features. |
| slog frontend (`sloghandler`) | 733 ns / 7 allocs | Standardizing on `slog.Default()` so dependencies' logs route through loglayer too. |

Sub-microsecond emission across the board. The sections below break down where each cost goes.

## Wrapping a third-party logger

### zerolog

`zerolog` is the fastest of the popular Go loggers; the question is what wrapping it costs.

| Setup | Time | Allocs | Bytes | Δ vs direct |
|---|---:|---:|---:|---:|
| Direct zerolog, simple message | 77 ns | 0 | 0 | — |
| LogLayer + zerolog, simple message | 142 ns | 1 | 16 | **+65 ns / +1 alloc** |
| Direct zerolog, three keyed fields | 138 ns | 0 | 0 | — |
| LogLayer + zerolog, struct metadata | 535 ns | 5 | 272 | **+397 ns / +5 allocs** |
| LogLayer + zerolog, map metadata | 713 ns | 8 | 544 | **+575 ns / +8 allocs** |

The simple-message overhead is one allocation: the `*LogBuilder` for the metadata/error chain plus dispatch through the wrapper transport.

Metadata costs more because LogLayer builds an intermediate map for the dispatched payload before handing it to zerolog. **Structs are cheaper than maps** because the encoder serializes the value directly. If you log the same shape repeatedly, declare a struct.

### zap

| Setup | Time | Allocs | Bytes | Δ vs direct |
|---|---:|---:|---:|---:|
| Direct zap, simple message | 370 ns | 0 | 0 | — |
| LogLayer + zap, simple message | 470 ns | 1 | 16 | **+100 ns / +1 alloc** |
| Direct zap, three keyed fields | 570 ns | 1 | 192 | — |
| LogLayer + zap, struct metadata | 1,018 ns | 5 | 320 | **+448 ns / +4 allocs** |
| LogLayer + zap, map metadata | 1,128 ns | 5 | 641 | **+558 ns / +4 allocs** |

The wrapper overhead has the same shape as zerolog's. zap's own dispatch is heavier (5× zerolog's floor), so the LogLayer tax is a smaller fraction of the total. If you're already on zap, the overhead is in the noise.

## Renderer transports

LogLayer ships two renderers that don't depend on a third-party logger:

- **`structured`** writes JSON-per-line via `github.com/goccy/go-json`. Recommended for production.
- **`console`** writes plain text in [logfmt](https://brandur.org/logfmt) form (`msg key=value key=value`). Useful for CI logs or terminal output without colors.

Both write into a pooled buffer and dispatch through the same plugin pipeline as wrapper transports.

| Setup | Time | Allocs | Bytes |
|---|---:|---:|---:|
| `structured`, simple message | 353 ns | 3 | 48 |
| `structured`, map metadata | 1,191 ns | 15 | 585 |
| `structured`, struct metadata | 1,634 ns | 16 | 633 |
| `console`, simple message | 155 ns | 3 | 48 |
| `console`, map metadata | 1,095 ns | 14 | 992 |
| `console`, struct metadata | 1,537 ns | 15 | 1,040 |

Reading these:

- **`console` beats `structured` on simple messages** (155 ns vs 353 ns) because it just writes the line. No JSON encoding.
- **`console` beats `structured` on map metadata** (1,095 ns vs 1,191 ns) because logfmt rendering is cheaper than JSON for shallow scalar payloads. Strings render bare when safe and quoted when they contain spaces or special chars; numbers and bools render directly. Nested values inside the data map (a `map[string]any` value, for instance) get JSON-encoded inline.
- **`structured` simple-message cost** is faster than wrapping zap (470 ns) and slower than wrapping zerolog (142 ns, hand-tuned JSON encoder). For most workloads the difference is invisible against the cost of actually shipping the bytes.
- **Struct metadata is slower than map** on both renderers because struct values go through a JSON roundtrip in `MetadataAsMap` before the renderer sees them. If you log the same struct shape repeatedly on a hot path and care about renderer cost, build the map directly instead.

## slog frontend

`integrations/sloghandler` makes every `slog.Info(...)` flow through the loglayer pipeline.

| Setup | Time | Allocs | Bytes |
|---|---:|---:|---:|
| **Baselines (no loglayer)** | | | |
| slog → no-op handler, simple msg | 229 ns | 0 | 0 |
| slog → no-op handler, three attrs | 384 ns | 3 | 144 |
| slog → stdlib JSON handler → discard, simple msg | 523 ns | 0 | 0 |
| slog → stdlib JSON handler → discard, three attrs | 928 ns | 3 | 144 |
| **LogLayer handler (no-op transport)** | | | |
| `slog.Info("msg")` | 733 ns | 7 | 664 |
| `slog.Info("msg", k1, v1, k2, v2, k3, v3)` | 1,394 ns | 15 | 1,224 |
| Persistent attrs via `slog.With(...)` | 1,137 ns | 12 | 1,080 |
| `slog.WithGroup("http").Info(...)` | 1,328 ns | 14 | 1,416 |
| LogValuer attr | 1,110 ns | 10 | 1,048 |
| `slog.InfoContext(ctx, ...)` | 744 ns | 7 | 664 |

How to read these numbers:

- **+229 ns is unavoidable.** Every `slog.Info(...)` captures `Record.PC` and builds a Record before any handler runs. That's slog's design.
- **+294 ns is JSON serialization** (523 − 229). The loglayer handler skips this because the transport is a no-op; in practice the `structured` transport adds comparable JSON marshalling.
- **+210 ns is the loglayer pipeline tax** on top of the JSON-emitting baseline (733 − 523). It buys the plugin pipeline, multi-transport fan-out, group routing, runtime level state, the typed `LogLine` testing capture, and source-info forwarding.
- **The 7-vs-0 alloc gap is structural.** The stdlib JSON handler reuses a `sync.Pool`-backed buffer per call; loglayer's assembled `Data` map can't be pooled because transports are allowed to retain it (the testing transport stores it directly; an async transport would hold it across goroutines). See [`AGENTS.md`](https://github.com/loglayer/loglayer-go/blob/main/AGENTS.md) "Performance: Attempted and Rejected".

The loglayer-native path (`log.Info("msg")` directly, no slog frontend) is ~41 ns / 1 alloc on the same hardware. If raw throughput on the message-emission path matters more than slog interop, call loglayer directly.

## Plugin pipeline overhead

This benchmark measures **framework dispatch cost**, not real plugin work. Three trivial hooks (a `DataHook` returning `{"tagged": true}`, a passthrough `LevelHook`, a passthrough `SendGate`):

| Setup | Time | Allocs | Bytes |
|---|---:|---:|---:|
| No plugins, simple message | 41 ns | 1 | 16 |
| Three trivial hooks, simple message | 433 ns | 5 | 688 |

The framework cost is per-hook params construction plus the `recover()` defer; LogLayer pre-indexes hook membership at registration time, so plugins that don't implement a given hook never run.

Real plugin cost is whatever the plugin does on top of dispatch:

- **redact**: dominated by deep-clone cost; scales with payload depth. Raw clone numbers in [`utils/maputil/cloner_test.go`](https://github.com/loglayer/loglayer-go/blob/main/utils/maputil/cloner_test.go).
- **sampling**: a few ns per emission; almost free for the throughput it saves.
- **oteltrace / datadogtrace**: a context value lookup plus a small `Data` return.

For mutation-heavy plugins, see [Creating Plugins → Performance: only clone if you mutate](/plugins/creating-plugins#performance-only-clone-if-you-mutate).

## Caller info (`Config.Source`)

`Config.Source.Enabled: true` captures file/line/function at every emission via `runtime.Caller` plus `runtime.FuncForPC`.

| Setup | Time | Allocs | Bytes | Δ vs off |
|---|---:|---:|---:|---:|
| Simple message, off | 41 ns | 1 | 16 | — |
| Simple message, on | 636 ns | 6 | 648 | **+595 ns / +5 allocs** |
| Map metadata, off | 235 ns | 4 | 448 | — |
| Map metadata, on | 891 ns | 9 | 1,080 | **+656 ns / +5 allocs** |

The added cost is constant across emission shapes and dominated by `runtime.FuncForPC().Name()` materializing the function-name string plus the heap-allocated `*Source`. Leave `Source.Enabled` off in throughput-sensitive code.

The slog handler ([`integrations/sloghandler`](/integrations/sloghandler)) forwards `slog.Record.PC` for free, since slog itself captures the PC regardless. The handler's hot path is comparable to the Source-on path above.

## When the overhead matters

LogLayer's overhead is **per-call cost**, not per-byte. If your service does I/O between log calls (an HTTP request, a database query, a goroutine wakeup), the budget is dominated by serialization, not dispatch. The wrapper overhead is invisible.

Cases where it becomes visible:

- A tight loop emitting millions of logs per second: pre-aggregate, [sample](/plugins/sampling), or gate behind `IsLevelEnabled` and skip the call entirely.
- A latency-sensitive hot path where every nanosecond is budgeted: gate behind `IsLevelEnabled`.

For typical web services, batch jobs, and CLI tools, the dispatch cost is in the noise.

## Reproducing locally

```sh
go test -bench=. -benchmem -run=^$ -benchtime=500ms .
```

`bench_test.go` covers the loglayer-internal benchmarks (renderers, plugins, source). Per-transport benchmarks live next to each transport:

```sh
go test -bench='Zerolog' -benchmem -run=^$ -benchtime=500ms ./transports/zerolog
```
