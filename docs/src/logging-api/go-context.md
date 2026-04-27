---
title: Go Context (context.Context)
description: Attach a context.Context to log entries so transports can extract trace IDs, span context, deadlines, and other request-scoped values.
---

# Go Context

LogLayer can attach a per-call `context.Context` to a log entry. Transports receive it via `TransportParams.Ctx` and can extract anything carried by the context: trace IDs, span context, deadlines, request-scoped values.

This is distinct from [`WithFields`](/logging-api/fields), which manages a persistent key/value bag on the logger. `WithCtx` is per-call only and never persists.

## Attaching context

```go
import "context"

func handle(ctx context.Context, log *loglayer.LogLayer) {
    log.WithCtx(ctx).Info("request received")
}
```

`WithCtx` returns a `*LogBuilder` so it chains with the existing builder API:

```go
log.WithCtx(ctx).
    WithMetadata(loglayer.Metadata{"path": r.URL.Path}).
    WithError(err).
    Error("handler failed")
```

Passing nil to `WithCtx` is a no-op.

## Reading context in a transport

Transports see the context as `params.Ctx`. They can extract trace IDs, deadlines, or any other context value:

```go
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
    if !t.ShouldProcess(params.LogLevel) {
        return
    }
    if params.Ctx != nil {
        if span := trace.SpanFromContext(params.Ctx); span.SpanContext().IsValid() {
            // attach trace_id / span_id to the entry
        }
    }
    // ...
}
```

`params.Ctx` is nil when the entry was emitted without `WithCtx`. Always check before using.

## Why per-call only?

`context.Context` carries deadlines, cancellation signals, and request-scoped values. None of those make sense to persist across multiple log calls (a deadline that fired three logs ago, a cancelled request you're still emitting for).

`WithFields` (the other one) is for persistent application data that should appear on every log: request ID, service name, user ID. Different problem, different API.

## Use the Raw entry for context

If you're constructing log entries via [`Raw`](/logging-api/raw), set `Ctx` on the `RawLogEntry`:

```go
log.Raw(loglayer.RawLogEntry{
    LogLevel: loglayer.LogLevelInfo,
    Messages: []any{"replayed"},
    Ctx:      ctx,
})
```

## Embedding the logger in a context

LogLayer also supports the inverse pattern: store the logger *inside* a `context.Context` so downstream code can extract it without threading it through every function signature. This is useful for HTTP/gRPC middleware that wants every request to log with request-scoped context (request ID, user ID, etc.) without changing every handler's signature.

### Middleware: attach a per-request logger

```go
func loggingMiddleware(base *loglayer.LogLayer) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            reqLog := base.Child()
            reqLog.WithFields(loglayer.Fields{
                "requestId": r.Header.Get("X-Request-ID"),
                "method":    r.Method,
                "path":      r.URL.Path,
            })

            ctx := loglayer.NewContext(r.Context(), reqLog)
            next.ServeHTTP(w, r.WithFields(ctx))
        })
    }
}
```

### Handler: extract the logger from context

```go
func handle(w http.ResponseWriter, r *http.Request) {
    log := loglayer.FromContext(r.Context())
    if log == nil {
        log = defaultLogger // fall back to a package-level logger
    }
    log.Info("handling request")
}
```

If your middleware is guaranteed to attach a logger, use `MustFromContext`:

```go
func handle(w http.ResponseWriter, r *http.Request) {
    log := loglayer.MustFromContext(r.Context())  // panics if missing
    log.Info("handling request")
}
```

This trades safety for ergonomics: the panic surfaces a misconfiguration immediately rather than letting logs silently disappear.

### Distinct from WithCtx

These two APIs use the word "context" for different things:

- `WithCtx(ctx)` attaches a `context.Context` to a single log entry so transports can read its values (trace IDs, deadlines).
- `NewContext(ctx, log)` / `FromContext(ctx)` stores the logger *inside* a `context.Context` for downstream retrieval.

You can combine them:

```go
log := loglayer.FromContext(r.Context())
log.WithCtx(r.Context()).Info("served")  // emits with both: logger from ctx, ctx values forwarded to transports
```
