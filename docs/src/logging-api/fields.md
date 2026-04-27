---
title: Fields
description: Persistent, keyed data attached to every log entry from a logger.
---

# Logging with Fields

Fields are data that should appear on every log entry from a logger: request IDs, user info, session data, anything that identifies the unit of work in progress. They are the opposite of [metadata](/logging-api/metadata), which is per-call.

## Adding Fields

`WithFields` returns a **new logger** with the given key/value pairs merged in. The receiver is unchanged. This matches the convention used by zerolog, zap, slog, logrus, and other Go logging libraries.

```go
log = log.WithFields(loglayer.Fields{
    "requestId": "abc-123",
    "userId":    "user_456",
})

log.Info("Processing request")
log.Warn("User quota exceeded")
```

Both subsequent calls include `requestId` and `userId`. By default the keys are merged at the root of the data object:

```json
{
  "msg": "Processing request",
  "requestId": "abc-123",
  "userId":    "user_456"
}
```

`loglayer.Fields` is a type alias for `map[string]any`.

::: warning Assign the result
`WithFields` returns a new logger; the original is untouched. If you discard the return value, the new logger is dropped on the floor and the original keeps emitting without your fields:

```go
log.WithFields(loglayer.Fields{"k": "v"})  // ❌ result discarded, log unchanged
log.Info("oops")                            // emits without "k"
```

The compiler doesn't catch this (no error return). Always assign:

```go
log = log.WithFields(loglayer.Fields{"k": "v"})  // ✅
log.Info("ok")                                    // emits with "k"
```
:::

## Per-Request Loggers

The shape above is the pattern for HTTP handlers, workers, and anything else where you want request-scoped fields without leaking across concurrent operations:

```go
var serverLog = loglayer.New(...) // shared across all handlers

func handler(w http.ResponseWriter, r *http.Request) {
    reqLog := serverLog.WithFields(loglayer.Fields{
        "requestId": r.Header.Get("X-Request-ID"),
    })
    reqLog.Info("handling request") // includes requestId
}
```

`reqLog` is goroutine-local; `serverLog` is unchanged. Concurrent handlers each get their own derived logger. If you do this in every handler, look at [`integrations/loghttp`](/integrations/loghttp) which wraps it as one-line middleware.

## Calling WithFields Multiple Times

Each call returns a logger that inherits the previous logger's fields:

```go
log = log.WithFields(loglayer.Fields{"a": 1})
log = log.WithFields(loglayer.Fields{"b": 2})
// Logger now carries {a: 1, b: 2}
```

Passing nil or an empty map returns a clone with no additions.

## Nesting Fields Under a Single Key

By default field keys merge at the root. To nest them under one key, set `FieldsKey`:

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    FieldsKey: "fields",
})

log = log.WithFields(loglayer.Fields{"requestId": "abc"})
log.Info("ok")
```

```json
{
  "msg": "ok",
  "fields": { "requestId": "abc" }
}
```

## Reading Current Fields

```go
fields := log.GetFields()
// returns a shallow copy; mutating it does not affect the logger
```

## Clearing Fields

`ClearFields` returns a new logger with the given keys removed. With no arguments, all fields are cleared.

```go
// Remove all
log = log.ClearFields()

// Remove specific keys
log = log.ClearFields("requestId")
log = log.ClearFields("requestId", "userId")
```

Chains compose because each method returns the new logger:

```go
log = log.WithFields(loglayer.Fields{"a": 1, "b": 2, "c": 3}).
    ClearFields("a")
log.Info("only b and c remain")
```

## Muting Fields

`MuteFields` and `UnmuteFields` mutate the logger in place (they're treated as setup-time admin toggles, not per-request operations):

```go
log.MuteFields()    // skip fields in emit
log.UnmuteFields()  // re-enable
```

Or set it on the config:

```go
log := loglayer.New(loglayer.Config{
    Transport:  structured.New(structured.Config{}),
    MuteFields: true,
})
```

## Fields vs Metadata

Both attach structured data to logs. The difference is scope:

- `WithFields`: returns a logger that carries the given fields on every subsequent emission.
- `WithMetadata`: attached to a single log entry only.

Fields are keyed (`map[string]any`) because they support keyed operations: merge, clear-by-key, copy on `Child()`. Metadata is `any` because each log entry is a one-shot payload. See [Metadata](/logging-api/metadata).

## Combining with Metadata and Errors

<!--@include: ./_partials/combining-example.md-->

## Child Loggers and Fields

`Child()` is the no-additions form of `WithFields`: it returns an independent clone with the parent's fields shallow-copied. Use `Child()` when you want isolation without adding fields, `WithFields(...)` when you also want to add some.

```go
log = log.WithFields(loglayer.Fields{"shared": "value"})
child := log.WithFields(loglayer.Fields{"child_only": "x"})

log.Info("parent")  // includes "shared" only
child.Info("child") // includes "shared" and "child_only"
```

See [Child Loggers](/logging-api/child-loggers).

## Thread Safety

Every method on `*loglayer.LogLayer` is safe to call from any goroutine, including concurrently with emission. `WithFields`, `ClearFields`, `Child`, and `WithPrefix` return a new logger; the receiver is unchanged. Level toggling, transport changes, and mute toggles can all run live without any coordination on your side.

See the full [thread-safety contract](https://github.com/loglayer/loglayer-go/blob/main/AGENTS.md#thread-safety) for the per-method breakdown.
