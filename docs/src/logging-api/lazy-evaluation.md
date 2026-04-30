---
title: Lazy Evaluation
description: "Defer expensive Fields computation until the log entry is actually emitted."
---

# Lazy Evaluation

::: info Credit
This feature is adapted from [LogTape's lazy evaluation](https://logtape.org/manual/lazy). Thank you to the LogTape team for answering questions around its implementation!
:::

`loglayer.Lazy(fn)` returns a `*LazyValue`, an opaque wrapper around a callback that's invoked at log emit time rather than when the field is attached. Use it for `WithFields` values that are expensive to compute and shouldn't be paid for on entries the level filter drops.

```go
func Lazy(fn func() any) *LazyValue
const LazyEvalError = "[LazyEvalError]"
```

`fn` may be nil; a nil callback panics on call, the recover substitutes `LazyEvalError`, and the rest of the entry still emits.

```go
import "runtime"

log = log.WithFields(loglayer.Fields{
    "service": "api",                                 // static value
    "heap_kb": loglayer.Lazy(func() any {             // lazy value
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        return m.HeapAlloc / 1024
    }),
})

log.Info("starting work") // service: "api", heap_kb: 512
// ... allocate a lot ...
log.Info("done")          // service: "api", heap_kb: 12480 (fresh)
```

Static and lazy values mix freely in the same `Fields` literal. Each emission re-runs every lazy callback the logger holds. If the level is disabled, callbacks never run.

## Where it works

`Lazy` is recognized only as the **direct value of a `Fields` key** (or `RawLogEntry.Fields` key). A `*LazyValue` nested inside another value (a map, slice, or struct field) is not resolved.

## Concurrency

Callbacks may be invoked concurrently from multiple goroutines if the same `*LogLayer` is shared and used to log on different goroutines. **Callback bodies must be safe for concurrent invocation.** Use `atomic` operations, a `sync.Mutex`, or pure functions.

## Memory

A `func() any` is a closure; whatever it captures stays alive as long as the holding logger does. Be deliberate about what your callback closes over: capturing a large request body or an entire response struct holds those values until the logger goes out of scope. For long-lived loggers, prefer reading state through pointers or accessor functions over capturing it.

Child loggers inherit the wrapper, not the resolved value, so each emit through a child re-runs the callback.

## Errors: `LazyEvalError`

If the callback panics, the panic is recovered, the value emitted in the entry is the placeholder constant `loglayer.LazyEvalError` (`"[LazyEvalError]"`), and the rest of the entry is sent normally so other fields aren't lost.

```go
if v == loglayer.LazyEvalError {
    // a Lazy callback panicked here
}
```

## Performance

Loggers without a `*LazyValue` attached pay only one atomic load per emission to check the cached flag. Attaching a lazy value to `Fields` adds a per-emission cost: a copy of the `Fields` map plus the callback's own work.

```go
// no lazy
log.WithFields(loglayer.Fields{
    "requestId": "abc-123",
    "service":   "api",
}).Info("done")

// lazy: same shape, one field is deferred
log.WithFields(loglayer.Fields{
    "requestId": "abc-123",
    "computed":  loglayer.Lazy(func() any { return "computed-value" }),
}).Info("done")
```

Measured on a noop transport (3-run averages):

| Shape | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| `WithFields` (no lazy) | 180 | 352 | 3 |
| `WithFields` with one lazy field | 345 | 688 | 5 |
| Δ | +165 | +336 | +2 |

The delta is the cost of copying the persistent `Fields` map every emission. The wrapper has to stay in place for re-evaluation on the next emit, so the framework can't mutate the original. If a hot path doesn't need the deferred computation, derive a separate logger (`log.WithoutFields("computed")`) so the dominant paths pay nothing.

---

Plugin authors writing a `FieldsHook` should know that `OnFieldsCalled` receives the raw `*LazyValue`; see [Creating Plugins](/plugins/creating-plugins#fieldshook).
