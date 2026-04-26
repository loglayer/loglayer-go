---
title: Child Loggers
description: Cloning a logger with isolated fields and level state.
---

# Child Loggers

`Child()` returns a new `*loglayer.LogLayer` that inherits the parent's configuration, transports, fields, and level state, and can be mutated independently.

```go
child := log.Child()
```

## What Gets Inherited

| Aspect             | Inherited?           | Mutations propagate?    |
|--------------------|----------------------|-------------------------|
| Transports         | Yes (same instances) | No (transport list copy)|
| `Config`           | Yes (value copy)     | No                      |
| Fields map         | Yes (shallow copy)   | No                      |
| Level state        | Yes (cloned)         | No                      |
| Prefix             | Yes (via config)     | No                      |

The child holds a reference to the same transport instances, they're shared. Everything else is copied so the child can be modified without touching the parent.

## Use Cases

### Per-request loggers

`WithFields` already returns a new logger, so the per-request pattern is just one assignment:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    reqLog := serverLog.WithFields(loglayer.Fields{
        "requestId": r.Header.Get("X-Request-ID"),
        "method":    r.Method,
        "path":      r.URL.Path,
    })

    reqLog.Info("handling request")
    // ... pass reqLog into business logic ...
    reqLog.Info("handled")
}
```

The parent `serverLog` is untouched, the next request gets a fresh derived logger. `Child()` is the no-additions form if you want isolation without adding fields.

### Per-subsystem loggers via `WithPrefix`

`WithPrefix` is `Child()` plus a prefix override:

```go
authLog := log.WithPrefix("[auth]")
dbLog   := log.WithPrefix("[db]")
```

Each subsystem can mutate its own fields, prefix, or level state without affecting the others.

### Tightening levels in one branch

```go
debugLog := log.Child()
debugLog.SetLevel(loglayer.LogLevelDebug)
```

The original `log` keeps its threshold; the child runs at debug for one code path.

## Fields Isolation

Each derived logger has its own fields map (shallow copy from the parent at derive time). Modifications never bleed.

```go
log = log.WithFields(loglayer.Fields{"shared": "value"})
child := log.WithFields(loglayer.Fields{"child_only": "x"})

log.Info("parent")  // {"msg": "parent", "shared": "value"}
child.Info("child") // {"msg": "child", "shared": "value", "child_only": "x"}
```

Note that the copy is **shallow**: if a field value is a map or slice, the parent and child point at the same underlying data. Don't mutate field values in place; replace the whole entry by deriving a new logger.

## Level Isolation

```go
log.SetLevel(loglayer.LogLevelInfo)
child := log.Child()

child.DisableLevel(loglayer.LogLevelInfo)

log.Info("emitted")    // parent: info enabled
child.Info("dropped")  // child: info disabled
```

## When Not to Use Child

If you want the same logger everywhere with no per-call divergence, just pass the original. `Child()` is for the moments you genuinely need an isolated copy.
