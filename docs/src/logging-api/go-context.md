---
title: Go Context (context.Context)
description: Bind a context.Context to a logger so transports and plugins can extract trace IDs, span context, deadlines, and other request-scoped values.
---

# Go Context

LogLayer can attach a `context.Context` to log entries. Transports and plugins that opt in can extract anything the context carries: trace IDs, span context, deadlines, request-scoped values.

`WithCtx` works two ways depending on the receiver, mirroring the [`WithGroup`](/logging-api/groups) pattern:

- **`(*LogLayer).WithCtx(ctx)`** returns a derived logger with the context **bound** to every subsequent emission. This is the recommended pattern for per-request handlers.
- **`(*LogBuilder).WithCtx(ctx)`** attaches the context to a **single emission only**. Useful as an override on a logger that already has a different context bound.

## Binding to a logger (recommended)

```go
import "context"

func handle(ctx context.Context, base *loglayer.LogLayer) {
    log := base.WithCtx(ctx) // bind once
    log.Info("entry received")
    log.WithMetadata(loglayer.Metadata{"step": 1}).Info("step done")
    log.Error("something failed")
    // every emission carries ctx
}
```

::: warning Assign the result
`(*LogLayer).WithCtx` returns a new logger; without assignment nothing is bound and transports/plugins see no context.

```go
log.WithCtx(r.Context())          // ❌ no assignment, ctx not bound
log = log.WithCtx(r.Context())    // ✅ persistent on logger
```

The builder-level `(*LogBuilder).WithCtx(ctx).Info(...)` form doesn't have this trap because the builder is single-use, but it only attaches for one emission.
:::

Inside an HTTP handler, the [`loghttp` middleware](/integrations/loghttp) does this automatically: the per-request logger from `loghttp.FromRequest(r)` already has `r.Context()` bound, so handlers just write `loghttp.FromRequest(r).Info(...)` and any plugin reading `params.Ctx` gets the request context.

```go
func handler(w http.ResponseWriter, r *http.Request) {
    log := loghttp.FromRequest(r)
    log.Info("processing")        // already bound to r.Context()
    log.Info("calling downstream") // same
}
```

The pattern works for any code path with a context:

- gRPC interceptors: bind `stream.Context()` (or unary's first arg) to the per-RPC logger.
- Background workers: bind the worker's cancellation context so log-trace correlation reflects the actual unit of work.
- Database / cache calls: extract the parent span from the call's context if your plugin reads it.

## Per-call override

The builder still has `WithCtx` for the rare cases where you want to override the bound context for a single emission:

```go
log := base.WithCtx(rootCtx)        // bound

log.Info("uses rootCtx")
log.WithCtx(otherCtx).Info("uses otherCtx, just this call")
log.Info("back to rootCtx")
```

Useful when you create a child span mid-handler and want the next log to reference it:

```go
childCtx, span := tracer.StartSpanFromContext(rootCtx, "subop")
log.WithCtx(childCtx).Info("doing subop") // child span's IDs land in this entry
span.Finish()
log.Info("back to root span")
```

## Use the Raw entry for context

If you're constructing entries via [`Raw`](/logging-api/raw), set `Ctx` on the `RawLogEntry`. It overrides the logger's bound context for that entry:

```go
log.Raw(loglayer.RawLogEntry{
    LogLevel: loglayer.LogLevelInfo,
    Messages: []any{"replayed"},
    Ctx:      ctx,
})
```

## Lifetime concerns

Binding a context to a long-lived logger can outlive the context's intended scope. Two things to be aware of:

- **Cancelled contexts.** If you bind a request context and the request finishes (deadline, cancellation), plugins reading `ctx.Err()` will see `context.Canceled`. Most plugins should be fine with this; some (e.g. plugins that gate dispatch on `ctx.Err() == nil`) explicitly use it as a signal.
- **Garbage retention.** A bound context might transitively reference request-scoped data. Don't bind a request context to a long-lived (e.g. package-global) logger or you'll keep that data alive. The `loghttp` middleware sidesteps this by scoping the bound logger to the request.

For background workers: bind the worker's own cancellation context, not the parent's.

## Embedding the logger in a context

LogLayer also supports the inverse pattern: store the logger *inside* a `context.Context` so downstream code can extract it. This is useful for HTTP/gRPC middleware (the [`loghttp` middleware](/integrations/loghttp) does this for you).

```go
ctx = loglayer.NewContext(ctx, reqLog) // middleware

log := loglayer.FromContext(ctx)        // handler — returns nil if absent
log := loglayer.MustFromContext(ctx)    // panics if absent
```

These two APIs use "context" for different things:

- `WithCtx(ctx)` binds a `context.Context` to a logger so transports/plugins read its values (trace IDs, deadlines).
- `NewContext(ctx, log)` / `FromContext(ctx)` stores the logger *inside* a `context.Context` for downstream retrieval.

The `loghttp` middleware does both: it derives a per-request logger, binds `r.Context()` to it, and also stores the logger in `r.Context()` for handlers to retrieve.
