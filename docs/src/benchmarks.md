---
title: Benchmarks
description: How much does LogLayer cost on top of the underlying logger?
---

# Benchmarks

## TL;DR

**LogLayer is slower than using the underlying logger directly.** That's the trade-off for a unified API across loggers and the ability to swap backends without rewriting log sites.

Concretely, for a simple log entry, **LogLayer adds about 40 to 90 nanoseconds and one allocation** on top of the underlying high-performance logger (zerolog, zap, phuslu, charmlog). For metadata, the gap is wider (a few hundred nanoseconds and a few allocations) because LogLayer has to translate from `any` to the underlying library's typed field calls.

If you're already pinned to a single logger and your hot path is sub-microsecond, use that logger directly. For everything else (most application logging), the overhead is invisible compared to the work the application is doing, and the API consistency is worth it.

LogLayer's own work (assembling the entry, dispatching to transports) is **36 ns and 1 alloc** with a no-op transport. Anything beyond that is the underlying logger's cost.

The full numbers are below. Run them yourself with:

```sh
go test -bench=. -benchmem -run=^$ -benchtime=1s .
```

## What does LogLayer cost on top of my current logger?

Each row is one log call writing to a discard sink. **Direct** is the underlying library used straight; **Wrapped** is LogLayer with the matching transport in front of it. **Cost** is what wrapping adds.

### Simple message

```go
// Wrapped (any LogLayer transport)
log.Info("user logged in")

// Direct equivalents (every library has its own idiomatic syntax):
zerologLog.Info().Msg("user logged in")
zapLog.Info("user logged in")
phusluLog.Info().Msg("user logged in")
logrusLog.Info("user logged in")
charmLog.Info("user logged in")
```

| Library  | Direct       | Wrapped      | LogLayer cost |
|----------|--------------|--------------|---------------|
| zerolog  |  73 ns / 0 a | 114 ns / 1 a |  +41 ns, +1 alloc |
| zap      | 311 ns / 0 a | 396 ns / 1 a |  +85 ns, +1 alloc |
| phuslu   | 136 ns / 0 a | 181 ns / 1 a |  +45 ns, +1 alloc |
| charmlog | 865 ns / 11 a| 952 ns / 13 a|  +87 ns, +2 allocs |
| logrus   |1590 ns / 19 a|1790 ns / 23 a| +200 ns, +4 allocs |

### Metadata (3 fields)

This row is not strictly apples-to-apples. **Direct** uses each library's typed field API; **Wrapped** uses LogLayer's `WithMetadata(loglayer.Metadata{...})`. The map path forces the wrapper to translate from `any` to typed field calls, so the gap is wider than it looks.

```go
// Wrapped (any LogLayer transport)
log.WithMetadata(loglayer.Metadata{
    "id":    42,
    "name":  "Alice",
    "email": "alice@example.com",
}).Info("user logged in")

// Direct equivalents:
zerologLog.Info().
    Int("id", 42).
    Str("name", "Alice").
    Str("email", "alice@example.com").
    Msg("user logged in")

zapLog.Info("user logged in",
    zap.Int("id", 42),
    zap.String("name", "Alice"),
    zap.String("email", "alice@example.com"),
)

logrusLog.WithFields(logrus.Fields{
    "id":    42,
    "name":  "Alice",
    "email": "alice@example.com",
}).Info("user logged in")

charmLog.Info("user logged in",
    "id", 42, "name", "Alice", "email", "alice@example.com",
)
```

| Library  | Direct (typed) | Wrapped (map)  | LogLayer cost |
|----------|----------------|----------------|---------------|
| zerolog  |  136 ns /  0 a |  620 ns /  7 a | +484 ns, +7 allocs |
| zap      |  526 ns /  1 a |  980 ns /  4 a | +454 ns, +3 allocs |
| phuslu   |  187 ns /  0 a |  495 ns /  3 a | +308 ns, +3 allocs |
| charmlog | 1757 ns / 18 a | 2284 ns / 26 a | +527 ns, +8 allocs |
| logrus   | 2883 ns / 29 a | 3246 ns / 33 a | +363 ns, +4 allocs |

### Metadata (struct)

LogLayer hands the struct to the transport unchanged. Wrappers either reflect into it (zerolog/zap/phuslu/charmlog) or JSON-roundtrip it (renderers). Closer to apples-to-apples than the map row above.

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Wrapped (any LogLayer transport)
log.WithMetadata(User{ID: 42, Name: "Alice", Email: "alice@example.com"}).
    Info("user logged in")
```

| Library  | Wrapped struct |
|----------|----------------|
| phuslu   |  464 ns / 2 allocs |
| zerolog  |  469 ns / 4 allocs |
| zap      |  898 ns / 4 allocs |
| charmlog | 1516 ns / 18 allocs |
| logrus   | 2594 ns / 28 allocs |

## What if I'm starting from scratch?

If you don't have an existing logger to wrap, pick a [renderer transport](/transports/). Each writes the entry directly without a third-party library.

```go
// Same call site for every renderer; only the Config differs:
log.Info("user logged in")
log.WithMetadata(loglayer.Metadata{"k": "v"}).Info("served")
```

| Renderer            | Simple message | With map metadata |
|---------------------|----------------|-------------------|
| console             |  145 ns /  3 allocs | 1386 ns / 15 allocs |
| pretty (NoColor)    |  563 ns /  8 allocs | 1939 ns / 26 allocs |
| testing             |  465 ns /  2 allocs |  600 ns /  4 allocs |
| structured          | 1324 ns / 17 allocs | 2253 ns / 25 allocs |

Quick guidance:

- **Local dev:** [pretty](/transports/pretty). Color costs almost nothing extra; the table shows it without color so you can see the formatter cost on its own.
- **Production:** [structured](/transports/structured), or wrap [zerolog](/transports/zerolog) / [zap](/transports/zap) for higher throughput.
- **Tests:** [testing](/transports/testing). It's allocation-light and exposes typed fields you can assert on.
- **Pipes / fixtures:** [console](/transports/console). Zero deps, fastest of the renderers.

## When does the overhead actually matter?

For most applications, **never**. A few hundred nanoseconds and a handful of allocations per log entry is invisible alongside any database query, HTTP call, or template render.

The two cases where it matters:

1. **Sub-microsecond hot loops.** Per-event logging with a budget of nanoseconds. Use the underlying library directly in the loop; use LogLayer everywhere else.
2. **Allocation-sensitive paths.** If you have GC-pause SLOs, the map-metadata path will allocate. Pass a struct (LogLayer doesn't touch it) or use the underlying library's typed field API directly.

## LogLayer's own cost (no I/O)

These numbers isolate LogLayer's pure overhead by routing entries to a no-op transport. Anything you add on top (a real wrapper, a renderer) adds its own cost.

```go
// Simple message
log.Info("user logged in")

// With metadata (map)
log.WithMetadata(loglayer.Metadata{
    "id": 42, "name": "Alice", "email": "alice@example.com",
}).Info("user logged in")

// With metadata (struct)
log.WithMetadata(User{ID: 42, Name: "Alice", Email: "alice@example.com"}).
    Info("user logged in")

// With persistent fields (set once, included on every subsequent log)
log.WithFields(loglayer.Fields{"requestId": "abc-123", "service": "api"})
log.Info("request handled")

// With error
log.WithError(err).Error("operation failed")
```

| Operation               | ns/op | B/op | allocs/op |
|-------------------------|------:|-----:|----------:|
| Simple message          |    36 |   16 | 1 |
| With metadata (map)     |   193 |  352 | 3 |
| With metadata (struct)  |    70 |   64 | 2 |
| With persistent fields  |   242 |  352 | 3 |
| With error              |   339 |  704 | 6 |

The struct case is faster than the map case because LogLayer hands the struct through unchanged; the map allocates one map per call. See [Metadata](/logging-api/metadata) for why.

## Thread Safety

**Concurrent log emission is safe.** Multiple goroutines may call any log method (`Info`, `WithMetadata`, `WithError`, `Raw`, etc.) on the same `*loglayer.LogLayer` simultaneously. Verified under `-race` in `concurrency_test.go`.

**Mutating the logger after construction is NOT safe to call concurrently with emission.** Methods that change logger state (`WithFields`, `ClearFields`, `AddTransport`, `RemoveTransport`, `WithFreshTransports`, `SetLevel`, `EnableLevel`, `DisableLevel`, `EnableLogging`, `DisableLogging`, `MuteFields`, `UnmuteFields`, `MuteMetadata`, `UnmuteMetadata`) are intended for setup time. They mutate without locking; calling them while another goroutine is emitting is a data race.

If you need to swap configuration at runtime while logs are flying, build a new `*LogLayer` (or call `Child()`) on the new state and atomic-swap a pointer at your call site. LogLayer doesn't dictate how that swap happens; the standard `sync/atomic.Pointer[loglayer.LogLayer]` pattern works.

`Child()` itself is safe to call concurrently with emission on the parent (it returns a new logger; doesn't mutate the receiver). The returned child is independent and follows the same rules.

## Methodology

All benchmarks log a single `Info` entry per iteration to a custom `discardWriter` (a no-op `io.Writer` that returns the byte count without writing anywhere). We use this rather than `io.Discard` because `charmbracelet/log` detects `io.Discard` and skips its formatting pipeline entirely, which would understate its real cost by ~300x.

Each benchmark has a counterpart:

- **Direct/&ast;** uses the library straight, no LogLayer involved.
- **Wrapped/&ast;** runs the same payload through `loglayer.New(loglayer.Config{Transport: <wrapper>})`.
- **Render/&ast;** uses LogLayer with a self-contained renderer.
- **Loglayer/&ast;** uses a no-op transport with no I/O at all (the table just above).

::: tip Hardware
Numbers above are from a single AMD EPYC 7413 / Go 1.26 run. Treat them as relative, not absolute. Run on your own hardware before quoting.
:::

## Reproducing

```sh
git clone https://github.com/loglayer/loglayer-go
cd loglayer-go
go test -bench=. -benchmem -run=^$ -benchtime=1s .
```

A single library:

```sh
go test -bench='Zerolog' -benchmem -run=^$ .
```

Just the core overhead:

```sh
go test -bench='^BenchmarkLoglayer_' -benchmem -run=^$ .
```

Track regressions over time with [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```sh
go install golang.org/x/perf/cmd/benchstat@latest

go test -bench=. -benchmem -run=^$ -count=10 . > old.txt
# ... make changes ...
go test -bench=. -benchmem -run=^$ -count=10 . > new.txt
benchstat old.txt new.txt
```
