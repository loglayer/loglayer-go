---
title: Cheat Sheet
description: One-page quick reference of the LogLayer for Go API.
---

# Cheat Sheet

## Construction

```go
import (
    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
)

// Panics on missing transport (typical setup).
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})

// Or explicit error handling (returns loglayer.ErrNoTransport on failure).
log, err := loglayer.Build(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})
```

## Log Levels

```go
log.Trace("...")
log.Debug("...")
log.Info("...")
log.Warn("...")
log.Error("...")
log.Fatal("...") // calls os.Exit(1) by default; set Config.DisableFatalExit to skip
```

Each method takes `...any`, joined with a space.

## Metadata

```go
// Map (loglayer.Metadata is a type alias for map[string]any)
log.WithMetadata(loglayer.Metadata{"id": 1}).Info("ok")

// Struct (transport handles serialization)
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")

// No message, log just the metadata
log.MetadataOnly(loglayer.Metadata{"status": "healthy"})
log.MetadataOnly(loglayer.Metadata{"status": "warn"}, loglayer.LogLevelWarn)
```

## Errors

```go
log.WithError(err).Error("failed")
log.ErrorOnly(err)
log.ErrorOnly(err, loglayer.ErrorOnlyOpts{LogLevel: loglayer.LogLevelFatal})
```

## Fields (persistent)

```go
// WithFields and ClearFields return a NEW logger; assign the result.
log = log.WithFields(loglayer.Fields{"requestId": "123"})

// Read it back
fields := log.GetFields()

// Remove specific keys (returns new logger)
log = log.ClearFields("requestId")

// Remove all (returns new logger)
log = log.ClearFields()

// Mute / unmute output without losing the data (mutates in place)
log.MuteFields().UnmuteFields()
```

## Go Context (per-call)

```go
import "context"

log.WithCtx(ctx).Info("request received")
```

Per-call only; surfaced to transports via `TransportParams.Ctx`. See [Go Context](/logging-api/go-context).

## Logger in a Go Context (zerolog-style)

```go
// Middleware: store request-scoped logger in ctx
ctx = loglayer.NewContext(ctx, reqLog)

// Handler: pull it back out
log := loglayer.FromContext(ctx)         // nil if not attached
log := loglayer.MustFromContext(ctx)     // panics if not attached
```

## Combining

```go
log.
    WithCtx(ctx).
    WithFields(loglayer.Fields{"requestId": "abc"}).
    WithMetadata(loglayer.Metadata{"duration_ms": 23}).
    WithError(err).
    Error("request failed")
```

`WithCtx`, `WithMetadata`, and `WithError` return a `*LogBuilder`; chain freely before terminating with `Info()` / `Warn()` / etc. Each builder is single-use.

## Child Loggers

```go
child := log.Child()                       // copy of fields + level state
prefixed := log.WithPrefix("[auth]")       // child with a prefix prepended
```

Mutations on the child do not affect the parent.

## Level Control

```go
log.SetLevel(loglayer.LogLevelWarn)        // enable warn and above
log.EnableLevel(loglayer.LogLevelDebug)    // turn one level on
log.DisableLevel(loglayer.LogLevelDebug)   // turn one level off
log.IsLevelEnabled(loglayer.LogLevelInfo)  // check
log.DisableLogging()                       // master kill switch
log.EnableLogging()
```

## Transport Management

```go
log.AddTransport(t)                    // append (replaces if same ID)
log.RemoveTransport("id")              // returns true if removed
log.WithFreshTransports(t1, t2)        // replace all
log.GetLoggerInstance("id")            // underlying logger from a transport
```

## Raw

```go
log.Raw(loglayer.RawLogEntry{
    LogLevel: loglayer.LogLevelInfo,
    Messages: []any{"already assembled"},
    Metadata: loglayer.Metadata{"k": "v"},
    Err:      err,
    Fields:   loglayer.Fields{"override": "ctx"}, // optional override
    Ctx:      ctx,                                 // optional Go context
})
```

## Levels

```go
loglayer.LogLevelTrace  // 10
loglayer.LogLevelDebug  // 20
loglayer.LogLevelInfo   // 30
loglayer.LogLevelWarn   // 40
loglayer.LogLevelError  // 50
loglayer.LogLevelFatal  // 60
```
