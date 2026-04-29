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
| `log.withContext({ ... })`            | `log.WithFields(loglayer.Fields{ ... })`          | **Renamed** (see below)              |
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

Per-call `context.Context` (the Go stdlib type, e.g. for trace IDs and deadlines) is attached separately via [`WithContext`](/logging-api/go-context):

```go
log.WithContext(ctx).Info("request received")
```

This concept doesn't exist in TS, since JavaScript doesn't have a comparable per-request context primitive.

## Constructor and error handling

TypeScript:

```ts
const log = new LogLayer({
  transport: new PinoTransport({ logger: pino() }),
});
```

Go has two constructors. The `New`/`Build` pair is the same pattern Go uses elsewhere when a misconfiguration is a programmer error you want to fail loudly on but still need a recoverable variant for env-driven setup:

<!--@include: ./_partials/constructors.md-->

## Errors

TypeScript errors carry a stack trace by default (the `Error` constructor in V8/JS engines). Go's `error` interface is just `interface { Error() string }`: no stack trace and no chain unless the error implementation provides one.

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

TypeScript's `@loglayer/transport-pino`, `@loglayer/plugin-redaction`, etc. are separate npm packages. The Go port follows the same model: the core is one module, and any transport/plugin with a third-party dep is its own module so consumers only pay for what they import.

| TypeScript                       | Go                                            |
|----------------------------------|-----------------------------------------------|
| `loglayer`                       | `go.loglayer.dev` (core + stdlib renderers)   |
| `@loglayer/transport-zerolog`    | `go.loglayer.dev/transports/zerolog`          |
| `@loglayer/transport-datadog`    | `go.loglayer.dev/transports/datadog`          |
| `@loglayer/integration-elysia`   | `go.loglayer.dev/integrations/loghttp` (etc.) |

`go get` each module you actually need; the dependency graph stays focused on whatever you imported.

## Plugins

The TypeScript plugin system maps directly, but the Go API is interface-based instead of object-with-methods. `loglayer.Plugin` is a one-method interface (`ID() string`); each lifecycle hook is its own optional interface that you implement on the same type.

| TypeScript hook (on `LogLayerPlugin`) | Go interface |
|---|---|
| `onContextCalled` | `loglayer.FieldsHook` |
| `onMetadataCalled` | `loglayer.MetadataHook` |
| `onBeforeDataOut` | `loglayer.DataHook` |
| `onBeforeMessageOut` | `loglayer.MessageHook` |
| `shouldSendToLogger` | `loglayer.SendGate` |
| (no equivalent) | `loglayer.LevelHook` |

For one-off single-hook plugins, use the adapter constructors:

```go
log.AddPlugin(loglayer.NewDataHook("tag-service", func(p loglayer.BeforeDataOutParams) loglayer.Data {
    return loglayer.Data{"service": "checkout"}
}))
```

The full set: `NewFieldsHook`, `NewMetadataHook`, `NewDataHook`, `NewMessageHook`, `NewLevelHook`, `NewSendGate`. For multi-hook plugins, declare a type implementing `Plugin` plus the relevant hook interfaces (the `plugins/redact` source is the canonical reference).

`plugins/redact` mirrors `@loglayer/plugin-redaction`. It supports key matching, regex value patterns, and json-tag-aware struct walking, all type-preserving:

```go
import "go.loglayer.dev/plugins/redact"

log.AddPlugin(redact.New(redact.Config{
    Keys:     []string{"password", "apiKey"},
    Patterns: []*regexp.Regexp{regexp.MustCompile(`^\d{16}$`)},
}))
```

See [Plugins](/plugins/) for the full lifecycle, hook ordering, and nil-return semantics. Third-party plugins can use [`utils/maputil`](https://pkg.go.dev/go.loglayer.dev/utils/maputil) for the same reflection-based deep-clone primitive that the redact plugin uses.

## Groups

Groups port directly. The TS `string | string[]` argument shape becomes Go variadic, and `null` for "clear filter" becomes a separate `ClearActiveGroups` method.

| TypeScript | Go |
|---|---|
| `log.withGroup('database')` | `log.WithGroup("database")` |
| `log.withGroup(['database', 'auth'])` | `log.WithGroup("database", "auth")` |
| `log.addGroup(name, { ... })` | `log.AddGroup(name, loglayer.LogGroup{...})` |
| `log.disableGroup(name)` / `enableGroup(name)` | `log.DisableGroup(name)` / `EnableGroup(name)` |
| `log.setGroupLevel(name, 'debug')` | `log.SetGroupLevel(name, loglayer.LogLevelDebug)` |
| `log.setActiveGroups(['db'])` | `log.SetActiveGroups("db")` |
| `log.setActiveGroups(null)` | `log.ClearActiveGroups()` |
| `log.getGroups()` | `log.GetGroups()` (shallow copy) |
| `LOGLAYER_GROUPS` env var auto-read | `Config.Routing.ActiveGroups: loglayer.ActiveGroupsFromEnv("LOGLAYER_GROUPS")` (explicit) |
| `ungroupedBehavior: 'all' \| 'none' \| string[]` | `Config.Routing.Ungrouped: loglayer.UngroupedRouting{Mode, Transports}` (typed enum) |

The Go port does not auto-read environment variables (libraries shouldn't); `ActiveGroupsFromEnv` is a helper you opt into. See [Groups](/logging-api/groups) for the full reference.

## Currently out of scope

These exist in TypeScript loglayer but are not yet implemented in the Go port:

- **Mixins**: the `useLogLayerMixin` augmentation pattern.
- **Context managers**: `LinkedContextManager`, `IsolatedContextManager`. Go's flat fields-as-map model covers most use cases.
- **Lazy evaluation**: `withMetadataLazy`, `withContextLazy`. Possible to add but the ergonomic story is weaker in Go.

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
