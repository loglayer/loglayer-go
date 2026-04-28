---
title: Benchmarks
description: Per-call cost of LogLayer for Go vs raw underlying loggers, plus the cost of plugins and metadata shapes.
---

# Benchmarks

LogLayer is a thin layer; it shouldn't materially change your service's CPU budget. This page compares each setup against a baseline so you can decide what's right for your workload.

::: tip How these were measured
`go test -bench=. -benchmem -run=^$` against a no-op writer, Go 1.25. Host CPU is an AMD EPYC 7413 (24 cores); the benchmark runs inside a VM with 16 cores assigned. Each benchmark exercises the entire dispatch path: emission method → plugin pipeline (zero plugins where unstated) → transport → discard writer.

Absolute numbers will vary on your hardware; the **relative cost** between setups is what's portable. Use these numbers to pick between abstraction strategies, not to predict end-to-end throughput.
:::

## Compared with raw zerolog

`zerolog` is the fastest of the popular Go loggers. If your call sites already use it, the question is what wrapping it with LogLayer costs.

| Setup | Time | Allocs | Bytes | vs direct |
|---|---:|---:|---:|---:|
| **Direct zerolog**, simple message | 74 ns | 0 | 0 | baseline |
| LogLayer wrapping zerolog, simple message | 118 ns | 1 | 16 | **+44 ns / +60%** |
| **Direct zerolog**, three keyed fields | 135 ns | 0 | 0 | baseline |
| LogLayer wrapping zerolog, struct metadata | 517 ns | 5 | 256 | **+382 ns / +283%** |
| LogLayer wrapping zerolog, map metadata | 688 ns | 8 | 528 | **+553 ns / +410%** |
| LogLayer's `transports/structured` (no zerolog at all), simple message | 332 ns | 3 | 48 | wrapping zerolog is ~2.8× faster |

The simple-message overhead is ~44 ns and one allocation per call: the `*LogBuilder` for the metadata/error chain, plus dispatch through the wrapper transport. For a service emitting 10k logs/sec, that's about 0.4 ms/sec of CPU.

Adding metadata costs more because LogLayer builds an intermediate map for the dispatched payload before handing it to zerolog. Structs avoid one of the allocations because the transport's encoder serializes the value directly. **If you log the same shape repeatedly, declaring a struct is meaningfully cheaper than building a fresh map every call.**

The last row makes a different point: LogLayer's own `structured` transport exists for projects that want a setup with no third-party logger. It writes JSON directly into a pooled buffer using `github.com/goccy/go-json` for value marshaling, which closes most of the gap to wrapping zerolog. If raw throughput is the priority, wrapping zerolog is still faster.

## Compared with raw zap

`zap` is a common alternative; numbers below show what each shape costs relative to direct zap.

| Setup | Time | Allocs | Bytes | vs direct |
|---|---:|---:|---:|---:|
| **Direct zap**, simple message | 360 ns | 0 | 0 | baseline |
| LogLayer wrapping zap, simple message | 444 ns | 1 | 16 | **+84 ns / +23%** |
| **Direct zap**, three keyed fields | 582 ns | 1 | 192 | baseline |
| LogLayer wrapping zap, struct metadata | 988 ns | 5 | 304 | **+406 ns / +70%** |
| LogLayer wrapping zap, map metadata | 1,100 ns | 5 | 625 | **+518 ns / +89%** |
| LogLayer's `transports/structured` (no zap at all), simple message | 332 ns | 3 | 48 | structured is ~1.3× faster than wrapping zap |

zap's own dispatch is heavier than zerolog's, so the relative cost of wrapping it is smaller (23% vs 60% on the simple-message path). Absolute LogLayer overhead is similar (~85 ns and one alloc); zap's higher floor makes the percentage shrink.

The same higher floor reverses the relationship with `transports/structured`: on the simple-message path, structured is now slightly faster than wrapping zap (332 ns vs 444 ns). If you wanted zap specifically for its API or its `zapcore` pipeline, the win is in features, not throughput.

## Plugin pipeline cost

A pipeline with three plugins (one `DataHook`, one `LevelHook`, one `SendGate`):

| Setup | Time | Allocs | Bytes |
|---|---:|---:|---:|
| LogLayer with no plugins, simple message (no-op transport) | 41 ns | 1 | 16 |
| LogLayer with three plugins, simple message (no-op transport) | 511 ns | 5 | 688 |

The framework pre-indexes hook membership at registration time, so plugins that don't implement a given hook never run. Read-only plugins don't allocate; mutating ones (the `redact` plugin's deep-clone, for instance) own most of the cost in a real pipeline.

For a heavy redaction pipeline, see [Creating Plugins → Performance: only clone if you mutate](/plugins/creating-plugins#performance-only-clone-if-you-mutate).

## When the overhead matters (and when it doesn't)

LogLayer's overhead is **per-call cost**, not per-byte. If your service does I/O between log calls (a single HTTP request, a database query, a goroutine wakeup), the budget you spend on logging is dominated by what you serialize, not by the dispatch overhead. The wrapper overhead is invisible.

The cases where the overhead becomes visible:

- A tight loop that logs millions of times per second (pre-aggregate, sample, or use a level filter).
- A latency-sensitive hot path where every nanosecond is budgeted (gate logs behind `IsLevelEnabled` and skip the call entirely).

For typical web services, batch jobs, and CLI tools, the framework cost is in the noise.

## Reproducing locally

```sh
go test -bench=. -benchmem -run=^$ -benchtime=500ms .
```

`bench_test.go` covers every wrapper transport plus the framework-internal benchmarks. To compare a single library:

```sh
go test -bench='Zerolog' -benchmem -run=^$ -benchtime=500ms .
```
