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

| Setup | Time | Allocs | Bytes | vs direct zerolog |
|---|---:|---:|---:|---:|
| **Direct zerolog**, simple message | 74 ns | 0 | 0 | baseline |
| LogLayer wrapping zerolog, simple message | 118 ns | 1 | 16 | **+44 ns / +60%** |
| **Direct zerolog**, three keyed fields | 135 ns | 0 | 0 | baseline |
| LogLayer wrapping zerolog, struct metadata | 517 ns | 5 | 256 | **+382 ns / +283%** (vs three keyed fields) |
| LogLayer wrapping zerolog, map metadata | 688 ns | 8 | 528 | **+553 ns / +410%** (vs three keyed fields) |
| LogLayer's `transports/structured` (no zerolog), simple message | 332 ns | 3 | 48 | **+258 ns / +349%** |

The simple-message overhead from wrapping is ~44 ns and one allocation per call: the `*LogBuilder` for the metadata/error chain, plus dispatch through the wrapper transport. For a service emitting 10k logs/sec, that's about 0.4 ms/sec of CPU.

Adding metadata costs more because LogLayer builds an intermediate map for the dispatched payload before handing it to zerolog. Structs avoid one of the allocations because the transport's encoder serializes the value directly. **If you log the same shape repeatedly, declaring a struct is meaningfully cheaper than building a fresh map every call.**

The last row offers a different choice: LogLayer's own `structured` transport (no third-party logger involved) at 332 ns. Compared with the **wrapping-zerolog row above (118 ns)**, the zerolog-backed setup is ~2.8× faster on simple messages because zerolog hand-tunes its JSON encoder. The standalone `structured` transport writes JSON directly into a pooled buffer using `github.com/goccy/go-json`, which closes most of the gap but doesn't beat zerolog. If raw throughput is the priority and you don't mind the dependency, wrap zerolog. If you want a setup with no third-party logger, `structured` is the answer.

## Compared with raw zap

`zap` is a common alternative; numbers below show what each shape costs relative to direct zap.

| Setup | Time | Allocs | Bytes | vs direct zap |
|---|---:|---:|---:|---:|
| **Direct zap**, simple message | 360 ns | 0 | 0 | baseline |
| LogLayer wrapping zap, simple message | 444 ns | 1 | 16 | **+84 ns / +23%** |
| **Direct zap**, three keyed fields | 582 ns | 1 | 192 | baseline |
| LogLayer wrapping zap, struct metadata | 988 ns | 5 | 304 | **+406 ns / +70%** (vs three keyed fields) |
| LogLayer wrapping zap, map metadata | 1,100 ns | 5 | 625 | **+518 ns / +89%** (vs three keyed fields) |
| LogLayer's `transports/structured` (no zap), simple message | 332 ns | 3 | 48 | **−28 ns / −8%** |

zap's own dispatch is heavier than zerolog's, so the relative cost of wrapping it is smaller (23% vs 60% on the simple-message path). Absolute LogLayer overhead is similar (~85 ns and one alloc); zap's higher floor makes the percentage shrink.

Note the last row: LogLayer's own `structured` transport (332 ns) is faster than wrapping zap (444 ns) on simple messages. The standalone JSON path beats zap once you account for the wrapper overhead. If you wanted zap specifically for its API or its `zapcore` pipeline, the win is in features, not throughput.

## Plugin pipeline cost

A pipeline with three plugins (one `DataHook`, one `LevelHook`, one `SendGate`):

| Setup | Time | Allocs | Bytes |
|---|---:|---:|---:|
| LogLayer with no plugins, simple message (no-op transport) | 41 ns | 1 | 16 |
| LogLayer with three plugins, simple message (no-op transport) | 511 ns | 5 | 688 |

The framework pre-indexes hook membership at registration time, so plugins that don't implement a given hook never run. Read-only plugins don't allocate; mutating ones (the `redact` plugin's deep-clone, for instance) own most of the cost in a real pipeline.

For a heavy redaction pipeline, see [Creating Plugins → Performance: only clone if you mutate](/plugins/creating-plugins#performance-only-clone-if-you-mutate).

## Caller info (`Config.Source`)

`Config.Source.Enabled: true` captures file/line/function at every emission via `runtime.Caller` plus `runtime.FuncForPC`. The cost is real and not negligible on hot paths.

| Setup | Time | Allocs | Bytes |
|---|---:|---:|---:|
| Simple message, Source off | 40 ns | 1 | 16 |
| Simple message, Source on | **660 ns** | **6** | **648** |
| Map metadata, Source off | 252 ns | 4 | 448 |
| Map metadata, Source on | **889 ns** | **9** | **1080** |

The added cost (~620 ns and 5 allocations) is constant across emission shapes; it's dominated by `runtime.FuncForPC().Name()` materializing the function-name string and the heap-allocated `*Source`. Leave `Source.Enabled` off in throughput-sensitive code and rely on transport-level rendering plus inline metadata.

The slog Handler ([integrations/sloghandler](/integrations/sloghandler)) forwards `slog.Record.PC` for free, since slog itself captures the PC regardless. The handler's hot path is comparable to the Source-on path above.

## slog Handler hot path

`integrations/sloghandler` translates a `*slog.Logger` call into a loglayer dispatch. Two baselines anchor the cost so the handler's overhead is meaningful in absolute and relative terms:

| Setup | Time | Allocs | Bytes | Notes |
|---|---:|---:|---:|---|
| **Baseline: slog → no-op handler**, simple msg | 225 ns | 0 | 0 | slog's own frontend cost (Record construction, attr handling, dispatch into Handle). The floor for any `slog.Info(...)` call. |
| **Baseline: slog → no-op handler**, three attrs | 385 ns | 3 | 144 | |
| **Baseline: slog → stdlib JSON handler → discard**, simple msg | 510 ns | 0 | 0 | What `slog.Info` costs through the stdlib JSON handler. The number you'd see using slog directly without loglayer. |
| **Baseline: slog → stdlib JSON handler → discard**, three attrs | 928 ns | 3 | 144 | |
| `slog.Info("msg")` via loglayer handler (noop transport) | 730 ns | 7 | 664 | **+220 ns / +7 allocs over the JSON baseline.** |
| `slog.Info("msg", k1, v1, k2, v2, k3, v3)` | 1358 ns | 15 | 1224 | **+430 ns / +12 allocs over the JSON baseline.** |
| Persistent attrs via `slog.With(...)` | 1110 ns | 12 | 1080 | |
| `slog.WithGroup("http").Info(...)` | 1290 ns | 14 | 1416 | |
| LogValuer attr | 1070 ns | 10 | 1048 | |
| `slog.InfoContext(ctx, ...)` (no plugins) | 730 ns | 7 | 664 | ctx forwarding is free vs the simple-message path. |

Reading this table:

- **+225 ns is unavoidable.** Every `slog.Info(...)` call captures `Record.PC` and builds a Record before any handler runs. That's slog's design, not ours.
- **+285 ns is the cost of JSON serialization** in the stdlib handler (510 - 225). The loglayer handler skips this because the transport is a no-op; in practice, the `structured` transport adds comparable JSON marshalling.
- **+220 ns is the loglayer pipeline tax** on top of an equivalent JSON-emitting setup. It buys you: the plugin pipeline (redact, oteltrace, datadogtrace, sampling, custom hooks), multi-transport fan-out, group routing, runtime level state, the typed `LogLine` testing capture, and the source-info forwarding.

The 7-vs-0 alloc gap is structural: the stdlib JSON handler reuses a `sync.Pool`-backed buffer per call; loglayer's assembled `Data` map can't be pooled because transports are allowed to retain it (the testing transport stores it directly; an async transport would hold it across goroutines). That's a documented contract trade-off; see [`AGENTS.md`](https://github.com/loglayer/loglayer-go/blob/main/AGENTS.md) "Performance: Attempted and Rejected".

For comparison, the loglayer-native path (`log.Info("msg")` directly, no slog frontend) is ~40 ns / 1 alloc on the same hardware: the slog frontend itself accounts for most of the cost on the slog path. If raw throughput on the message-emission path matters more than slog interop, call loglayer directly.

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
