---
title: Metadata
description: "Per-log structured data: maps, structs, or any value."
---

# Logging with Metadata

Metadata attaches structured data to a single log entry. Unlike [fields](/logging-api/fields), it does not persist. Once the entry is emitted, the metadata is discarded. For a side-by-side comparison of `Fields`, `Metadata`, and `Data` (the third concept that surfaces in plugins), see [Fields, Metadata, and Data](/concepts/data-shapes).

## `loglayer.Metadata`: the canonical map shape

The most common payload is a string-keyed bag of values, so `loglayer` exports a named type `Metadata` for `map[string]any`. Map literals (`loglayer.Metadata{...}`) work like the underlying map for indexing, range, `len`, etc.; the named type lets the compiler distinguish `Metadata` from `Fields` (the persistent-on-logger shape) and `Data` (the assembled output transports see).

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

The core logger does **zero conversion**: your value is handed to the transport as-is. The transport decides how to render it. See each transport's page for exact shape.

## Building the Value First

`WithMetadata` accepts any Go value, so you don't have to construct the literal at the call site. Build the value (map, struct, pointer, slice, scalar) ahead of time and pass the variable in:

```go
// Map built first, passed as a variable
md := loglayer.Metadata{"userId": 42, "action": "login"}
md["browser"] = r.Header.Get("User-Agent")
log.WithMetadata(md).Info("user logged in")

// Struct built first, passed as a variable
evt := UserEvent{UserID: 42, Name: "Alice"}
log.WithMetadata(evt).Info("user logged in")

// Pointer works too; the transport's encoder dereferences when it serializes
log.WithMetadata(&evt).Info("user logged in")
```

This is useful when the payload is computed across several lines, populated conditionally, or reused across multiple log calls. The runtime behavior is identical to passing the literal inline.

The core never dereferences or copies the value: it stores `any` and hands it to the transport. Pointer-vs-value behavior is therefore a transport concern. Transports that use `encoding/json` (structured, http, datadog) and the wrappers that hand off to a JSON-aware logger (zap, zerolog, slog, logrus, charmlog, phuslu) all dereference pointers via the standard library or their own marshaler.

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

::: warning The map is not deep-copied
LogLayer doesn't clone the `Metadata` map you pass in. Mutating it after the call (e.g. reusing the same map for the next emission with a tweak) can bleed into the previous log when a transport retains the value. Build a fresh map per call, or treat the value as read-only once handed off.
:::

`MetadataFieldName` (set on a wrapper transport's config) only affects **non-map** metadata. Map metadata always flattens to root attributes; that's the whole point of using `loglayer.Metadata` for ad-hoc bags. For keyed data that should always nest under a fixed name, use the `Fields` API.

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

A typical JSON output:

```json
{
  "msg": "request handled",
  "method": "POST",
  "path": "/users",
  "duration_ms": 45
}
```

The exact shape (whether the struct flattens at the root or nests under a key) depends on the transport. See each transport's page for its rendering rules, or [Creating Transports: Handling `any` Metadata](/transports/creating-transports#handling-any-metadata) for the underlying placement policies and the helpers that implement them.

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
log.MetadataOnly(loglayer.Metadata{"cpu": "90%"}, loglayer.MetadataOnlyOpts{LogLevel: loglayer.LogLevelWarn})
```

The default level is `Info`. Passing `nil` is a no-op.

## Muting Metadata

Suppress metadata in output without removing the call sites. The toggle is `atomic.Bool` so concurrent reads are safe, but flipping mid-emission can interleave (some entries see pre-toggle, others post). Treat it as a setup-time admin toggle.

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

## Mutating metadata with a plugin

If you want to redact, rewrite, or filter metadata globally before it reaches a transport, register a plugin with an `OnMetadataCalled` hook. See [Plugins](/plugins/) and the built-in [redact plugin](/plugins/redact).
