---
title: About LogLayer for Go
description: Why LogLayer for Go exists and what it gives you over reaching for zerolog or zap directly.
---

# Introduction

Go has plenty of good logging libraries: `zerolog`, `zap`, `slog`, `logrus`. They all do the basics (`Info`, `Warn`, `Error`) but each one wires structured fields and errors differently. Once your app is committed to a library, switching means rewriting every log site.

LogLayer for Go is a thin layer that sits on top of those libraries. You write your application code against one fluent API; the underlying library is a configuration choice.

::: tip Coming from TypeScript?
This is the Go port of [`loglayer`](https://loglayer.dev) for TypeScript. The mental model and API shape map directly. See [For TypeScript Developers](/for-typescript-developers) for the full convention map and the deliberate Go-specific differences (`Fields` instead of `context`, threading guarantees, error handling, module layout).
:::

```go
log.
    WithMetadata(loglayer.Metadata{"userId": "1234"}).
    WithError(errors.New("something went wrong")).
    Error("user action failed")
```

```json
{
  "msg": "user action failed",
  "userId": "1234",
  "err": { "message": "something went wrong" }
}
```

## Bring Your Own Logger

LogLayer is designed to wrap an existing logger. The library ships transports for [zerolog](/transports/zerolog) and [zap](/transports/zap), plus first-party [structured JSON](/transports/structured), [console](/transports/console), and [test](/transports/testing) transports. Writing your own transport is [a single interface](/transports/creating-transports) with four methods.

```go
log := loglayer.New(loglayer.Config{
    Transport: zerolog.New(zerolog.Config{Logger: &myZerolog}),
})
```

Want to switch to zap? Replace the transport. Application code is untouched.

## Consistent API

You don't need to remember whether the library expects fields-then-message or message-then-fields, or which method takes a struct vs a map.

```go
// With LogLayer, same call shape regardless of backend
log.WithMetadata(loglayer.Metadata{"some": "data"}).Info("my message")

// Without LogLayer, every library wants something different
zerologLogger.Info().Interface("some", "data").Msg("my message")
zapLogger.Info("my message", zap.Any("some", "data"))
slog.Info("my message", "some", "data")
```

## Separation of Fields, Metadata, and Errors

LogLayer distinguishes three kinds of structured data, each with a clear scope:

| Type        | Method            | Scope                          | Purpose                                     |
|-------------|-------------------|--------------------------------|---------------------------------------------|
| **Fields**  | `WithFields()`    | Persistent across all logs     | Request IDs, user info, session data        |
| **Metadata**| `WithMetadata()`  | Single log entry only          | Event-specific details, durations, counts  |
| **Errors**  | `WithError()`     | Single log entry only          | An `error` value, serialized for output     |

```go
log.
    WithFields(loglayer.Fields{"requestId": "abc-123"}). // persists
    WithMetadata(loglayer.Metadata{"duration": 150}).         // this log only
    WithError(errors.New("timeout")).                       // this log only
    Error("Request failed")
```

```json
{
  "msg": "Request failed",
  "requestId": "abc-123",
  "duration": 150,
  "err": { "message": "timeout" }
}
```

The benefit isn't just naming. Per-log metadata can never accidentally leak into future logs, errors are serialized consistently, and each type can be nested under a dedicated field via [configuration](/configuration).

See [fields](/logging-api/fields), [metadata](/logging-api/metadata), and [error handling](/logging-api/error-handling).

## Type-flexible Metadata

`WithMetadata` accepts `any`. Pass a map for ad-hoc fields, a struct for typed payloads, or a scalar; the transport decides how to render it.

```go
type User struct {
    ID    int    `json:"id"`
    Email string `json:"email"`
}

log.WithMetadata(User{ID: 7, Email: "alice@example.com"}).Info("user")
log.WithMetadata(loglayer.Metadata{"latency_ms": 23}).Info("served")
```

The structured and console transports merge maps at the root and JSON-roundtrip structs into root fields. The zerolog and zap transports merge maps at the root and place struct payloads under a configurable field. See each transport page for details.

## Multi-Transport Fan-out

A single `LogLayer` instance can dispatch each log entry to multiple transports.

```go
log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        console.New(console.Config{}),
        structured.New(structured.Config{Writer: jsonFile}),
    },
})

log.Info("user signed in") // both transports receive it
```

See [multi-transport support](/transports/multiple-transports).

## Easy Mocking

For tests that don't care about log output, `loglayer.NewMock()` returns a drop-in `*LogLayer` backed by a discard transport:

```go
log := loglayer.NewMock()
DoWork(log, "input") // silent
```

For tests that want to *assert on* what was logged, the [`testing` transport](/transports/testing) captures every entry into a mutex-safe library with typed `LogLine` fields:

```go
lib := &lltest.TestLoggingLibrary{}
log := loglayer.New(loglayer.Config{
    Transport: lltest.New(lltest.Config{Library: lib}),
})

log.WithMetadata(loglayer.Metadata{"k": "v"}).Info("msg")

line := lib.PopLine()
require.Equal(t, "msg", line.Messages[0])
require.Equal(t, "v", line.Metadata.(loglayer.Metadata)["k"])
```

See [Mocking](/logging-api/mocking) for the full pattern guide.

