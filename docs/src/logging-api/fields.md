---
title: Fields
description: Persistent, keyed data attached to every log entry from a logger.
---

# Logging with Fields

Fields are data that should appear on every log entry from a logger: request IDs, user info, session data, anything that identifies the unit of work in progress. They are the opposite of [metadata](/logging-api/metadata), which is per-call. For a side-by-side comparison of `Fields`, `Metadata`, and `Data` (the third concept that surfaces in plugins), see [Fields, Metadata, and Data](/concepts/data-shapes).

## Adding Fields

`WithFields` returns a **new logger** with the given key/value pairs merged in. The receiver is unchanged, matching zerolog, zap, slog, and logrus.

::: warning Assign the result
The compiler doesn't catch a discarded result. Always assign:

```go
log.WithFields(loglayer.Fields{"k": "v"})  // ❌ result discarded, log unchanged
log = log.WithFields(loglayer.Fields{"k": "v"})  // ✅
```

The same trap applies when handing the result to a function: `go runHandler(log)` drops the wrapper; pass `log.WithFields(...)` instead.
:::

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

::: warning The map is not deep-copied
LogLayer doesn't clone the `Fields` map you pass in. If you mutate it after `WithFields` returns, transports that retain the map (the testing transport, some async transports) will see the mutation. Treat the map as read-only after handing it off, or build a fresh one per call.
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

`WithoutFields` returns a new logger with the given keys removed. With no arguments, all fields are cleared.

```go
// Remove all
log = log.WithoutFields()

// Remove specific keys
log = log.WithoutFields("requestId")
log = log.WithoutFields("requestId", "userId")
```

Chains compose because each method returns the new logger:

```go
log = log.WithFields(loglayer.Fields{"a": 1, "b": 2, "c": 3}).
    WithoutFields("a")
log.Info("only b and c remain")
```

## Muting Fields

`MuteFields` and `UnmuteFields` mutate the logger in place. The state is `atomic.Bool` so concurrent reads from the dispatch path are safe, but flipping the toggle mid-emission can interleave: some entries see the pre-toggle state, others the post. Treat them as setup-time admin toggles. For a clean cutover, route through a feature flag or level toggle.

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

## Combining with Metadata and Errors

<!--@include: ./_partials/combining-example.md-->

## Mutating fields with a plugin

If you want to redact, rename, or otherwise rewrite fields globally before they're stored, register a plugin with an `OnFieldsCalled` hook. See [Plugins](/plugins/) and the built-in [redact plugin](/plugins/redact).
