---
title: HTTP Middleware (loghttp)
description: One-line HTTP middleware that derives a per-request logger and stuffs it into the request context.
---

# HTTP Middleware (loghttp)

`integrations/loghttp` is HTTP middleware that does the per-request logger derivation for you. Drop it into your router once at startup, and every handler downstream gets a logger pre-populated with `requestId`, `method`, `path`. The middleware also emits a "request completed" log line at the end of every request with status code, bytes written, and duration.

Mirrors the role that `hlog.NewHandler` plays for zerolog. Works with any `net/http`-compatible router: stdlib, chi, gorilla/mux, gin, echo, etc.

```sh
go get go.loglayer.dev/integrations/loghttp
```

## Basic Usage

```go
import (
    "net/http"

    "go.loglayer.dev"
    "go.loglayer.dev/integrations/loghttp"
    "go.loglayer.dev/transports/structured"
)

var log = loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/users", handler)

    http.ListenAndServe(":8080", loghttp.Middleware(log)(mux))
}

func handler(w http.ResponseWriter, r *http.Request) {
    log := loghttp.FromRequest(r)
    log.Info("looking up user")    // includes requestId, method, path
}
```

A request to `GET /users` produces:

```json
{"level":"info","time":"...","msg":"looking up user","requestId":"3f1a...","method":"GET","path":"/users"}
{"level":"info","time":"...","msg":"request completed","requestId":"3f1a...","method":"GET","path":"/users","status":200,"durationMs":2,"bytes":42}
```

## What the Middleware Does

For each incoming request:

1. Reads the request ID from `X-Request-ID`. Generates one (8 random bytes hex-encoded) if the header is absent.
2. Derives a per-request logger from the base logger via `WithFields`, attaching `requestId`, `method`, `path`.
3. Stores the per-request logger in `r.Context()` via `loglayer.NewContext`.
4. Wraps the response writer to capture status code and bytes written.
5. Calls `next.ServeHTTP(...)`.
6. Emits a "request completed" log line at the end with status, duration, and bytes in metadata.

The base logger is **never mutated**. Concurrent requests each get their own derived logger.

## Retrieving the Logger in Handlers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    log := loghttp.FromRequest(r)  // returns nil if middleware not applied
    // or:
    log := loghttp.MustFromRequest(r)  // panics if not applied
    log.Info("doing work")
}
```

`FromRequest(r)` is a thin wrapper around `loglayer.FromContext(r.Context())`.

## Options

```go
type Option func(*config)
```

All options are functional options passed to `Middleware(log, opts...)`.

### `WithRequestIDHeader(name string)`

Sets the HTTP header read for an incoming request ID. Default `"X-Request-ID"`.

```go
loghttp.Middleware(log, loghttp.WithRequestIDHeader("X-Trace-ID"))
```

### `WithRequestIDGenerator(fn func() string)`

Function called when no request ID header is present. Default: 8 random bytes hex-encoded.

```go
loghttp.Middleware(log, loghttp.WithRequestIDGenerator(func() string {
    return uuid.NewString()
}))
```

### `WithFieldNames(names FieldNames)`

Override the field keys. Empty values keep the default for that field.

```go
loghttp.Middleware(log, loghttp.WithFieldNames(loghttp.FieldNames{
    RequestID:  "trace_id",
    Status:     "http_status",
    DurationMs: "duration_ms",
}))
```

Default keys: `requestId`, `method`, `path`, `status`, `durationMs`, `bytes`.

### `WithStartLog(enabled bool)`

When true, emit a "request started" log line at the start of every request in addition to the "request completed" line. Default false to keep log volume low.

```go
loghttp.Middleware(log, loghttp.WithStartLog(true))
```

### `WithStatusLevels(fn func(status int) loglayer.LogLevel)`

Customize the log level for the completion log based on the response status code. Default:

| Status     | Level       |
|------------|-------------|
| 5xx        | `LogLevelError` |
| 4xx        | `LogLevelWarn`  |
| else       | `LogLevelInfo`  |

```go
loghttp.Middleware(log, loghttp.WithStatusLevels(func(status int) loglayer.LogLevel {
    if status >= 500 {
        return loglayer.LogLevelError
    }
    return loglayer.LogLevelInfo // demote 4xx to info
}))
```

### `WithExtraFields(fn func(*http.Request) loglayer.Fields)`

Attach additional fields to the per-request logger. Useful for tenant ID, user ID, trace ID extracted from headers or the URL path.

```go
loghttp.Middleware(log, loghttp.WithExtraFields(func(r *http.Request) loglayer.Fields {
    return loglayer.Fields{
        "tenant": r.Header.Get("X-Tenant-ID"),
        "userId": userIDFromAuth(r),
    }
}))
```

## Composing with Other Middleware

The middleware is shape `func(http.Handler) http.Handler`, the standard composition primitive in Go. Every router consumes it without adapters.

```go
// stdlib
http.Handle("/", loghttp.Middleware(log)(myHandler))

// chi
r := chi.NewRouter()
r.Use(loghttp.Middleware(log))

// gorilla/mux
r := mux.NewRouter()
r.Use(loghttp.Middleware(log))
```

## Optional Response Writer Interfaces

The middleware wraps `http.ResponseWriter` to capture status and bytes. The wrapper implements `Unwrap() http.ResponseWriter`, so handlers needing optional interfaces (`Flusher`, `Hijacker`, `Pusher`) should use `http.NewResponseController(w)` rather than type-asserting on the wrapper directly. This is the modern idiom (Go 1.20+).

```go
func sseHandler(w http.ResponseWriter, r *http.Request) {
    // Works through the wrapper via Unwrap.
    rc := http.NewResponseController(w)
    rc.Flush()
}
```

## Why This Exists

Without the middleware, every handler that wants per-request fields has to do this boilerplate:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    reqID := r.Header.Get("X-Request-ID")
    if reqID == "" { reqID = generateID() }
    reqLog := serverLog.WithFields(loglayer.Fields{
        "requestId": reqID,
        "method":    r.Method,
        "path":      r.URL.Path,
    })
    r = r.WithContext(loglayer.NewContext(r.Context(), reqLog))
    // ... wrap response writer for status capture ...
    // ... emit start/end logs with duration ...
    // actual handler logic
}
```

Easy to forget the `WithContext`. Easy to forget the response writer wrap. Easy to attach the wrong field set in different handlers. The middleware does it all once at registration so every handler is consistent.
