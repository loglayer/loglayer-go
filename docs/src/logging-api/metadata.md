---
title: Metadata
description: "Per-log structured data: maps, structs, or any value."
---

# Logging with Metadata

Metadata attaches structured data to a single log entry. Unlike [fields](/logging-api/fields), it does not persist. Once the entry is emitted, the metadata is discarded.

`WithMetadata` accepts **any** value. The core logger does no conversion; the transport decides how to serialize.

## Struct vs Map: pick the right shape

Two shapes are common. Use the one that matches your call site.

### Use a struct when the shape is fixed

If the same set of keys appears at this call site every time, declare a struct. The transport's encoder walks it directly, types are checked at compile time, and there's no map allocation per call.

```go
type RequestInfo struct {
    Method     string `json:"method"`
    Path       string `json:"path"`
    DurationMs int    `json:"duration_ms"`
}

log.WithMetadata(RequestInfo{
    Method:     "POST",
    Path:       "/users",
    DurationMs: 45,
}).Info("request handled")
```

```json
{"msg":"request handled","method":"POST","path":"/users","duration_ms":45}
```

This is the cheaper path on hot code: see [Benchmarks](/benchmarks) for the numbers (struct metadata is ~3 fewer allocations per emission than the map literal below).

### Use `loglayer.Metadata` when the shape varies

When you don't know which keys you'll have until runtime (conditional adds, varying domain values, ad-hoc bags), use `loglayer.Metadata` (a named alias for `map[string]any`):

```go
md := loglayer.Metadata{"userId": 42, "action": "login"}
if r.Header.Get("X-Debug") == "1" {
    md["browser"] = r.Header.Get("User-Agent")
}
log.WithMetadata(md).Info("user logged in")
```

The `loglayer.Metadata` named type lets the compiler distinguish it from `Fields` (persistent on the logger) and `Data` (the assembled output transports see). At runtime it is `map[string]any`; both these calls are identical:

```go
log.WithMetadata(loglayer.Metadata{"userId": 42}).Info("user")
log.WithMetadata(map[string]any{"userId": 42}).Info("user")
```

Prefer `loglayer.Metadata` throughout your code so the compiler can flag mix-ups with `Fields`.

::: warning The map is not deep-copied
LogLayer doesn't clone the map you pass to `WithMetadata`. Mutating it after the call (e.g. reusing the same map for the next emission with a tweak) can bleed into the previous log when a transport retains the value. Build a fresh map per call, or treat the value as read-only once handed off. Structs sidestep this entirely.
:::

`MetadataFieldName` (set on a wrapper transport's config) only affects **non-map** metadata; map metadata flattens to root attributes. The exact shape (struct flattens at the root vs. nests under a key) depends on the transport. See each transport's page for its rendering rules, or [Creating Transports → Handling `any` Metadata](/transports/creating-transports#handling-any-metadata) for the placement policies.

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
