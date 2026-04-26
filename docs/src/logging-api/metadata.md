---
title: Metadata
description: "Per-log structured data: maps, structs, or any value."
---

# Logging with Metadata

Metadata attaches structured data to a single log entry. Unlike [fields](/logging-api/fields), it does not persist. Once the entry is emitted, the metadata is discarded.

## `loglayer.Metadata`: the canonical map shape

The most common payload is a string-keyed bag of values, so `loglayer` exports a type alias `Metadata` for `map[string]any`. The two forms are 100% interchangeable at runtime; the alias just keeps call sites short.

```go
// Idiomatic
log.WithMetadata(loglayer.Metadata{"userId": 42, "action": "login"}).Info("user")

// Identical at runtime
log.WithMetadata(map[string]any{"userId": 42, "action": "login"}).Info("user")
```

Use `loglayer.Metadata` throughout your code unless you have a reason not to.

## The `any` Design

Although `Metadata` is the most common shape, `WithMetadata` accepts `any`. The transport decides how to serialize the value, so you can pass a struct, a slice, or anything else, and each transport renders it idiomatically.

```go
type User struct {
    ID    int    `json:"id"`
    Email string `json:"email"`
}

log.WithMetadata(User{ID: 7, Email: "alice@example.com"}).Info("user")
```

Per-transport handling:

- The [structured](/transports/structured) and [console](/transports/console) transports merge maps at the root and JSON-roundtrip structs into root fields.
- The [zerolog](/transports/zerolog) and [zap](/transports/zap) transports merge maps at the root and place struct payloads under a configurable field (default `"metadata"`).
- The [pretty](/transports/pretty) transport renders maps inline as `key=value` and JSON-roundtrips structs into the same shape.

The core logger does **zero conversion**: your value is handed to the transport as-is, so transports that can serialize structs natively (zerolog, zap) avoid an unnecessary roundtrip.

## Map Metadata

The common case. Keys are merged at the root by default:

```go
log.WithMetadata(loglayer.Metadata{
    "userId":  "123",
    "action":  "login",
    "browser": "Chrome",
}).Info("User logged in")
```

```json
{
  "msg": "User logged in",
  "userId": "123",
  "action": "login",
  "browser": "Chrome"
}
```

## Struct Metadata

Pass a struct directly. The transport handles serialization:

```go
type RequestInfo struct {
    Method   string `json:"method"`
    Path     string `json:"path"`
    Duration int    `json:"duration_ms"`
}

log.WithMetadata(RequestInfo{
    Method:   "POST",
    Path:     "/users",
    Duration: 45,
}).Info("request handled")
```

For the structured transport this becomes:

```json
{
  "msg": "request handled",
  "method": "POST",
  "path": "/users",
  "duration_ms": 45
}
```

For the zerolog transport (with default config), the struct lands under a `metadata` key:

```json
{ "message": "request handled", "metadata": { "method": "POST", ... } }
```

See each transport page for exact behavior.

## Replacing, Not Merging

Calling `WithMetadata` twice on the same builder **replaces** the value:

```go
log.WithMetadata(loglayer.Metadata{"a": 1}).
    WithMetadata(loglayer.Metadata{"b": 2}).
    Info("only b is attached")
```

This contrasts with `WithFields`, which merges. The reason is that metadata can be any value: there's no general "merge a struct into a map" operation, so the consistent rule is replace.

If you need to merge maps, do it before passing:

```go
combined := mergeMaps(m1, m2)
log.WithMetadata(combined).Info("ok")
```

## MetadataOnly

To log just metadata with no message:

```go
log.MetadataOnly(loglayer.Metadata{
    "status": "healthy",
    "memory": "512MB",
})

// or at a specific level
log.MetadataOnly(loglayer.Metadata{"cpu": "90%"}, loglayer.LogLevelWarn)
```

The default level is `Info`. Passing `nil` is a no-op.

## Muting Metadata

Suppress metadata in output without removing the call sites:

```go
log.MuteMetadata()    // skip metadata in emit
log.UnmuteMetadata()  // re-enable
```

Or via config:

```go
loglayer.New(loglayer.Config{
    Transport:    structured.New(structured.Config{}),
    MuteMetadata: true,
})
```

## Combining with Fields and Errors

<!--@include: ./_partials/combining-example.md-->
