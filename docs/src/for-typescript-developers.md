---
title: For TypeScript Developers
description: API and convention differences between LogLayer for TypeScript and the Go port.
---

# For TypeScript Developers

The Go port keeps the same mental model as the [TypeScript original](https://loglayer.dev): a fluent API on top of any underlying logger, with persistent fields, per-call metadata, and errors as first-class concepts. If you're already using `loglayer` in a TS service, the Go API will feel familiar.

This page covers the deliberate conventions and naming differences so you don't have to discover them by trial and error.

## API mapping

Methods are PascalCase in Go (per language convention) and camelCase in TypeScript:

| TypeScript                            | Go                                                | Notes                                |
|---------------------------------------|---------------------------------------------------|--------------------------------------|
| `new LogLayer({ ... })`               | `loglayer.New(loglayer.Config{ ... })`            | Function, not a class                |
| `log.withContext({ ... })`            | `log.WithFields(loglayer.Fields{ ... })`          | **Renamed** — see below              |
| `log.withMetadata({ ... })`           | `log.WithMetadata(loglayer.Metadata{ ... })`      | Same shape                           |
| `log.withError(err)`                  | `log.WithError(err)`                              | Same shape                           |
| `log.withPrefix(s)`                   | `log.WithPrefix(s)`                               | Same shape; returns a new logger     |
| `log.child()`                         | `log.Child()`                                     | Same shape                           |
| `log.info('msg')`                     | `log.Info("msg")`                                 | Variadic `...any` like TS varargs    |
| `log.errorOnly(err)` / `metadataOnly` | `log.ErrorOnly(err)` / `MetadataOnly`             | Same shape                           |
| `log.disableLogging()`                | `log.DisableLogging()`                            | Safe to call from any goroutine      |
| `MockLogLayer`                        | `loglayer.NewMock()`                              | Returns the concrete `*LogLayer`     |

## Why `Context` → `Fields`

The TS library calls the persistent key/value bag **context** (`withContext`, `contextFieldName`, `IContextManager`). In Go, `context` is a stdlib package (`context.Context`) used pervasively for request scoping, cancellation, and deadlines. Calling our concept `Context` would mean every reader has to figure out *which* context.

So Go uses **`Fields`**:

- `withContext` → `WithFields`
- `clearContext` → `WithoutFields`
- `muteContext` → `MuteFields`
- `contextFieldName` → `FieldsKey`
- The type alias `loglayer.Fields` is `map[string]any`, same as TS's loose object shape.

The behavior is identical; only the name changed.

Per-call `context.Context` (the Go stdlib type, e.g. for trace IDs and deadlines) is attached separately via [`WithCtx`](/logging-api/go-context):

```go
log.WithCtx(ctx).Info("request received")
```

This concept doesn't exist in TS, since JavaScript doesn't have a comparable per-request context primitive.

## Constructor and error handling

TypeScript:

```ts
const log = new LogLayer({
  transport: new PinoTransport({ logger: pino() }),
});
```

Go has two constructors. `New` panics on misconfiguration (typical idiom for setup-time errors); `Build` returns an `error` instead:

```go
log := loglayer.New(loglayer.Config{
    Transport: pretty.New(pretty.Config{}),
})

// Or, with explicit error handling:
log, err := loglayer.Build(loglayer.Config{
    Transport: pretty.New(pretty.Config{}),
})
```

Both report `loglayer.ErrNoTransport` if no transport is set.

## Errors

TypeScript errors carry a stack trace by default (the `Error` constructor in V8/JS engines). Go's `error` interface is just `interface { Error() string }` — no stack trace and no chain unless the error implementation provides one.

We recommend [`github.com/rotisserie/eris`](https://github.com/rotisserie/eris) for stack-trace-bearing errors. Its `ToJSON` plugs straight into LogLayer's `ErrorSerializer`:

```go
import "github.com/rotisserie/eris"

loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    ErrorSerializer: func(err error) map[string]any {
        return eris.ToJSON(err, true) // include stack trace
    },
})
```

See [Error Handling](/logging-api/error-handling) for the full reference.

## Threading

This is where Go and TS genuinely diverge. JavaScript runs on a single-threaded event loop; TS code never has true parallelism within a process. Go has goroutines and a real shared-memory threading model.

LogLayer for Go's contract:

- Every method on `*LogLayer` is safe to call from any goroutine, including concurrently with emission.
- `WithFields`, `WithoutFields`, `Child`, `WithPrefix` return a **new** logger; the receiver is unchanged. (This matches the convention used by zerolog, zap, slog, and logrus.) **Always assign the result**: `log = log.WithFields(...)`.
- Level mutators, transport mutators, and mute toggles are all safe to call live (e.g. operator-driven debug toggling via SIGUSR1, hot-reload of transport lists), with no special coordination on your side.

See the full [thread-safety contract](https://github.com/loglayer/loglayer-go/blob/main/AGENTS.md#thread-safety).

## Per-request loggers

The pattern is the same in both languages: derive a per-request logger and pass it down. The Go port ships first-class HTTP middleware in [`integrations/loghttp`](/integrations/loghttp) so this is one line at server setup:

```go
http.ListenAndServe(":8080", loghttp.Middleware(log, loghttp.Config{})(mux))
```

Inside a handler:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    log := loghttp.FromRequest(r) // includes requestId, method, path
    log.Info("processing")
}
```

## Module layout

TypeScript's `@loglayer/transport-pino`, `@loglayer/plugin-redaction`, etc. are separate npm packages. The Go port is a single Go module with sub-packages:

| TypeScript                       | Go                                            |
|----------------------------------|-----------------------------------------------|
| `loglayer`                       | `go.loglayer.dev` (the core)                  |
| `@loglayer/transport-zerolog`    | `go.loglayer.dev/transports/zerolog`          |
| `@loglayer/transport-datadog`    | `go.loglayer.dev/transports/datadog`          |
| `@loglayer/integration-elysia`   | `go.loglayer.dev/integrations/loghttp` (etc.) |

You install once (`go get go.loglayer.dev`) and import the sub-packages you need. Go's [lazy module loading](https://go.dev/ref/mod#lazy-loading) means transports you don't import don't add to your binary or `go.sum`.

## Plugins

The TypeScript plugin system maps directly. Where TS has a class implementing optional methods on the `LogLayerPlugin` interface, Go has a `loglayer.Plugin` struct with optional function fields. nil fields are skipped.

| TypeScript hook | Go field on `loglayer.Plugin` |
|---|---|
| `onContextCalled` | `OnFieldsCalled` |
| `onMetadataCalled` | `OnMetadataCalled` |
| `onBeforeDataOut` | `OnBeforeDataOut` |
| `onBeforeMessageOut` | `OnBeforeMessageOut` |
| `shouldSendToLogger` | `ShouldSend` |
| (no equivalent) | `TransformLogLevel` |

```go
log.AddPlugin(loglayer.Plugin{
    ID: "tag-service",
    OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
        return loglayer.Data{"service": "checkout"}
    },
})
```

Three convenience constructors for the common single-hook cases:

```go
log.AddPlugin(loglayer.MetadataPlugin("upper", fn))
log.AddPlugin(loglayer.FieldsPlugin("rename", fn))
log.AddPlugin(loglayer.LevelPlugin("promote", fn))
```

The first-party `plugins/redact` mirrors `@loglayer/plugin-redaction`. It supports key matching, regex value patterns, and json-tag-aware struct walking, all type-preserving:

```go
import "go.loglayer.dev/plugins/redact"

log.AddPlugin(redact.New(redact.Config{
    Keys:     []string{"password", "apiKey"},
    Patterns: []*regexp.Regexp{regexp.MustCompile(`^\d{16}$`)},
}))
```

See [Plugins](/plugins/) for the full lifecycle, hook ordering, and nil-return semantics. Third-party plugins can use [`utils/maputil`](https://pkg.go.dev/go.loglayer.dev/utils/maputil) for the same reflection-based deep-clone primitive that the redact plugin uses.

## Currently out of scope

These exist in TypeScript loglayer but are not yet implemented in the Go port:

- **Mixins** — the `useLogLayerMixin` augmentation pattern
- **Context managers** — `LinkedContextManager`, `IsolatedContextManager`. Go's flat fields-as-map model covers most use cases.
- **Lazy evaluation** — `withMetadataLazy`, `withContextLazy`. Possible to add but the ergonomic story is weaker in Go.
- **Group routing** — multi-target dispatch by group key.

If any of these are blockers for your use case, open an issue at [github.com/loglayer/loglayer-go](https://github.com/loglayer/loglayer-go).

## Quick reference

```go
// Equivalent of:
//   const log = new LogLayer({ transport: new PinoTransport({ logger: pino() }) });
//   log.withContext({ reqId: 'abc' });
//   log.withMetadata({ duration: 42 }).withError(err).info('did the thing');

import (
    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
)

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})
log = log.WithFields(loglayer.Fields{"reqId": "abc"})
log.WithMetadata(loglayer.Metadata{"duration": 42}).
    WithError(err).
    Info("did the thing")
```

Same JSON output shape; same mental model.
