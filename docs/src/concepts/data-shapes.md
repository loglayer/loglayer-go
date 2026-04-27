---
title: "Fields, Metadata, and Data"
description: "The three keyed-data concepts in LogLayer and which to use when."
---

# Fields, Metadata, and Data

LogLayer has three keyed-data concepts. They look similar (all three have the underlying type `map[string]any`) but each plays a distinct role in the dispatch pipeline. Picking the right one is mostly about scope — *who carries this, and for how long?*

## The three concepts

| Concept                 | Type alias            | Scope                                    | Set via                              | Use it for |
|-------------------------|-----------------------|------------------------------------------|--------------------------------------|-----------|
| **`loglayer.Fields`**   | `map[string]any` (named) | Persistent on the logger             | `WithFields`, `WithoutFields`        | Request IDs, user info, anything that should appear on every subsequent log from this logger. |
| **`loglayer.Metadata`** | `map[string]any` (named) | Per-call (one log entry)             | `WithMetadata` (also accepts any value, not just maps) | Per-event payload: counters, durations, the body of a request, structs that vary per call. |
| **`loglayer.Data`**     | `map[string]any` (named) | Per-call (assembled output to transport) | The framework builds it; plugins receive and may augment it via `OnBeforeDataOut` | The merged shape transports actually emit: fields, the serialized error, plus whatever plugins added. |

`Fields`, `Metadata`, and `Data` are **distinct named types** even though they share the same underlying `map[string]any`. The compiler will catch if you accidentally pass a `Metadata` value where `Fields` is expected. Map literals (`loglayer.Fields{...}`) and untyped `map[string]any` values are still assignable to any of them.

## Decision tree

- **Will this attribute appear on more than one log call?** Use `Fields`. Bind it once via `WithFields(loglayer.Fields{...})`, hand the resulting logger around, and every emission carries it.
- **Is this a one-off payload for a single log line?** Use `Metadata`. `WithMetadata` accepts any value, not just maps — pass a struct, a slice, an `int`, whatever fits. Maps flatten to root attributes in most transports; non-map values typically nest under `MetadataFieldName`.
- **You're writing a plugin?** You'll see `Data` in `BeforeDataOutParams.Data` and as the return type of `OnBeforeDataOut`. Treat it as the assembled output the next stage will see — return a partial map to merge in, or `nil` to add nothing.

## Common patterns

```go
// Bind persistent fields once on the per-request logger.
reqLog := log.WithFields(loglayer.Fields{
    "requestId": rid,
    "userId":    uid,
})

// Per-call metadata for a single emission.
reqLog.WithMetadata(loglayer.Metadata{
    "durationMs": elapsed.Milliseconds(),
    "status":     resp.StatusCode,
}).Info("request served")
```

```go
// WithMetadata accepts any value — a struct works too.
type usagePayload struct {
    BytesIn  int `json:"bytes_in"`
    BytesOut int `json:"bytes_out"`
}
log.WithMetadata(usagePayload{BytesIn: 1024, BytesOut: 4096}).Info("rpc done")
```

```go
// Plugin: read assembled Data, return a partial Data map to merge.
log.AddPlugin(loglayer.Plugin{
    ID: "add-region",
    OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
        return loglayer.Data{"region": "us-east-1"}
    },
})
```

## What goes where

A single emission can carry all three at once. The framework merges them in this order, so later stages can inspect (or override) what earlier stages produced:

1. The logger's persistent **fields** seed the assembled map.
2. The serialized error (if `WithError` was called) lands under `Config.ErrorFieldName` (default `"err"`).
3. Plugins' `OnBeforeDataOut` hooks run and may augment the **data**.
4. The transport receives `TransportParams{Data, Metadata, ...}` and decides how to render. Map metadata typically flattens to root; non-map metadata nests under `MetadataFieldName`.

For the dispatch order in detail and how each stage interacts with plugins and transports, see [Creating Plugins](/plugins/creating-plugins) and [Creating Transports](/transports/creating-transports).

## Why three named types and not one?

They serve genuinely different roles:

- **Fields** are **logger state**: they live on a `*LogLayer`, returned-new from `WithFields`. Confusing them with per-call metadata loses the whole point.
- **Metadata** is **caller intent**: "I'm emitting this entry and I want to attach this payload, just for this entry."
- **Data** is **framework state**: the assembled object on its way to a transport. Plugins observe and shape it.

Making them distinct named types means the compiler catches accidental misuse (passing a `Metadata` value to `WithFields` is now a build error). The runtime cost is zero — the underlying type is still `map[string]any` and ranging/indexing/`len` all work identically.
