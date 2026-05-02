---
title: Creating Transports
description: Implement the Transport interface to send LogLayer entries anywhere.
---

# Creating Transports

A transport is anything implementing four methods:

```go
type Transport interface {
    ID() string
    IsEnabled() bool
    SendToLogger(params loglayer.TransportParams)
    GetLoggerInstance() any
}
```

The `transport.BaseTransport` struct in this module handles `ID`, `IsEnabled`, and level filtering, embed it and you only need to implement `SendToLogger` and `GetLoggerInstance`.

## Minimal Example

```go
package mytransport

import (
    "fmt"
    "io"

    "go.loglayer.dev/v2"
    "go.loglayer.dev/v2/transport"
)

type Config struct {
    transport.BaseConfig
    Writer io.Writer
}

type Transport struct {
    transport.BaseTransport
    cfg Config
}

func New(cfg Config) *Transport {
    return &Transport{
        BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
        cfg:           cfg,
    }
}

func (t *Transport) ID() string                  { return t.BaseTransport.ID() }
func (t *Transport) IsEnabled() bool             { return t.BaseTransport.IsEnabled() }
func (t *Transport) GetLoggerInstance() any      { return nil }

func (t *Transport) SendToLogger(p loglayer.TransportParams) {
    if !t.BaseTransport.ShouldProcess(p.LogLevel) {
        return
    }

    fmt.Fprintf(t.cfg.Writer, "[%s] %v %v\n", p.LogLevel, p.Messages, p.Data)
}
```

That's the whole shape. From here it's a question of how you want to render the entry.

## What's in TransportParams

```go
type TransportParams struct {
    LogLevel LogLevel
    Messages []any   // raw message slice; the prefix is exposed separately on Prefix below
    Data     Data    // assembled fields + error map; nil when both are absent. Use len(Data) > 0 to check.
    Metadata any     // raw value passed to WithMetadata, your transport decides serialization
    Err      error
    Fields   Fields
    Ctx      context.Context // per-call WithContext value, or nil
    Groups   []string        // merged persistent + per-call WithGroup tags, or nil
    Schema   Schema           // resolved assembly shape: FieldsKey, MetadataFieldName, ErrorFieldName, SourceFieldName
    Prefix   string           // value attached via WithPrefix; empty when none was set
}
```

`Data` is the convenience map combining fields + error. `Metadata` is `any`, you choose how to render it. `Err` and `Fields` are also exposed raw if you want to inspect them directly. `Groups` is the merged set of persistent (`WithGroup` on the logger) and per-call (`WithGroup` on the builder) group tags for this entry; it's `nil` when no groups apply. `Schema` carries the keys the core wants used for placement (`FieldsKey`, `MetadataFieldName`, `ErrorFieldName`, `SourceFieldName`); read it before falling back to your transport's defaults.

## Handling `any` Metadata

`params.Metadata` is whatever the caller passed to `WithMetadata`: a map, a struct, a pointer, a slice, a scalar, or `nil`. Your transport picks a **placement policy** for each shape. The common choices:

- Flatten a map's keys at the root of the log object.
- JSON-roundtrip a struct so its fields surface as root keys.
- Nest a non-map value under a single `metadata`-style key.
- Hand the raw value to an attribute-aware backend and let it serialize.

The `transport` package exposes helpers that encode each policy. Reach for them before writing your own type switch.

| Helper | What it does | Use when |
|--------|--------------|----------|
| `transport.MetadataAsRootMap(v) (map[string]any, bool)` | Returns the map directly if `v` is `loglayer.Metadata` or `map[string]any`; otherwise `nil, false`. No allocation, no roundtrip. | Deciding whether to flatten or nest. The wrapper transports (zap, zerolog, slog, logrus, charmlog, phuslu) call this first, then nest non-map values under the metadata key supplied via `params.Schema.MetadataFieldName` (or a transport-specific default like `"metadata"` when unset). |
| `transport.MetadataAsMap(v) map[string]any` | Map fast path; non-map values are JSON-roundtripped into a map. Returns nil on roundtrip failure (channels, cycles, marshalers producing a non-object). | Renderers that flatten everything at the root, used by `structured` and (via `MergeFieldsAndMetadata`) by `console`. |
| `transport.MergeFieldsAndMetadata(p) map[string]any` | Combines `p.Data` and metadata into a single map. Honors `p.Schema.MetadataFieldName`: when set, the entire metadata value nests under that key; when empty, map metadata merges at root and non-map roundtrips via `MetadataAsMap` (dropped if the roundtrip fails). | Renderers that emit a single flat object. |
| `transport.MergeIntoMap(dst, data, metadata, metadataKey)` | Mutates `dst` in place. When `metadataKey` is non-empty, the entire metadata value nests under that key; when empty, map metadata merges at root and non-map lands raw under `"metadata"`. Encoders with access to `TransportParams` should pass `params.Schema.MetadataFieldName`. | Encoders that have already seeded `dst` with their own protocol fields (level, time, msg, ddsource, ...) and want to layer user data on top. Used by HTTP `JSONArrayEncoder` and Datadog. |
| `transport.FieldEstimate(p) int` | Counts the eventual root-level fields. | Pre-sizing slices/maps in attribute-style backends (zap, charmlog, otellog). |

### Picking a policy

The right choice depends on what the backend can render natively. Pick the section that matches your backend; each ends with a runnable example in the repo.

#### Renderer / "flatten" policy

Use this when your backend writes a flat shape like JSON-per-line or a terminal column ("key=value key=value"). Flatten everything to the root via `MergeFieldsAndMetadata`: map metadata merges in place, struct metadata is JSON-roundtripped, and slices / scalars are dropped (since they don't roundtrip into a map).

If silently dropping non-roundtrippable values is wrong for your audience, decide explicitly what to do with them:

- `structured`, `console`: drop them.
- `pretty`: fall back to a single `_metadata` key with a stringified value (so a human reader still sees something).
- `http`, `datadog`: nest them under a fixed `metadata` key.

Worked example: [`examples/custom-transport`](https://github.com/loglayer/loglayer-go/blob/main/examples/custom-transport/main.go).

#### Wrapper / "attribute-forwarding" policy

Use this when your backend already has an attribute API (zap's `zap.Any`, zerolog's `Event.Interface`, OTel's `KeyValue`, slog's `Attr`, ...). Read `params.Schema.MetadataFieldName` first: when non-empty, nest the raw value (map or otherwise) under that single key uniformly. Otherwise branch on metadata shape via `MetadataAsRootMap`:

- If it's a map, flatten each entry into its own attribute call.
- If it's not, forward the raw value as a single attribute under your transport's default key (typically `"metadata"`) and let the backend's marshaler render it natively. **Skip the JSON roundtrip**, the backend will encode the value at write time.

This is what every wrapper transport in the repo does (zap, zerolog, slog, logrus, charmlog, phuslu, OpenTelemetry).

Worked example: [`examples/custom-transport-attribute`](https://github.com/loglayer/loglayer-go/blob/main/examples/custom-transport-attribute/main.go).

#### Why two policies, not one

The built-in transports settle on whichever of the two matches their backend, so callers see consistent behavior: map metadata always flattens to root keys; struct metadata always renders idiomatically per transport. That contract is what the [Metadata page](/logging-api/metadata) advertises to users, and the helpers above are how it's enforced.

### Don't reinvent

Don't roll your own `metadataAsMap` unless your transport needs a placement policy these helpers don't already encode. The pretty transport's `_metadata` fallback is the only built-in example, and it's there because pretty is a human-readable renderer where dropping a slice silently is worse than showing it stringified.

## Reading `params.Ctx`

`params.Ctx` carries the `context.Context` the caller bound via [`WithContext`](/logging-api/go-context). It's `nil` when no context was attached. Use it when your transport needs to forward the context to a downstream library (OpenTelemetry, slog handlers, anything context-aware) or extract values from it (trace IDs, deadlines, request-scoped data).

```go
func (t *Transport) SendToLogger(p loglayer.TransportParams) {
    if !t.ShouldProcess(p.LogLevel) {
        return
    }
    ctx := p.Ctx
    if ctx == nil {
        ctx = context.Background() // for downstream calls that demand a non-nil ctx
    }
    t.downstream.WithContext(ctx).Log(p.LogLevel, transport.JoinMessages(p.Messages))
}
```

Two patterns built-in transports follow:

- **Wrapper transports forward the context** so the underlying library sees the same `context.Context` the caller bound. The slog wrapper passes `params.Ctx` to `slog.Logger.LogAttrs`; the OpenTelemetry transport hands it to the OTel logs SDK so the active span's trace/span IDs land on the record automatically. Any wrapper around a context-aware backend should do this.
- **Self-contained renderers usually ignore it.** Pretty, structured, and console don't read context values themselves; that's a [plugin's](/plugins/creating-plugins) job. If you find yourself extracting trace IDs in a transport, prefer writing a plugin and pairing it with the transport: the plugin runs once per entry and feeds every transport, while transport-side extraction repeats per transport and bypasses the dispatch-time hook ordering.

If your transport extracts values from the context (rather than just forwarding it), test that path with a context that carries a sentinel value and assert the transport surfaced it.

## Reading `params.Groups`

`params.Groups` carries the merged set of group tags for this entry: persistent ones from `log.WithGroup(...)` on the logger, plus per-call ones from `log.WithGroup(...)` on the builder. Persistent tags come first; per-call tags are appended (deduped). The slice is `nil` when no groups apply.

Routing decisions consume groups before the transport sees them: the dispatch layer uses `Groups` to pick which transports an entry goes to, and the slice arrives only after that decision has been made. Read it when your transport ships to a group-aware aggregator that wants the tags as part of the wire payload.

```go
func (t *Transport) SendToLogger(p loglayer.TransportParams) {
    if !t.ShouldProcess(p.LogLevel) {
        return
    }
    payload := t.buildPayload(p)
    if len(p.Groups) > 0 {
        payload["groups"] = p.Groups
    }
    t.send(payload)
}
```

`Groups` is shared with the dispatching `*LogLayer`. Don't mutate the slice in place. If you need to reorder, dedupe further, or filter, copy first.

## Reading `params.Prefix`

`params.Prefix` is the value attached via `WithPrefix` on the emitting logger (or set on `Config.Prefix` at construction), exposed verbatim so transports can render it independently from the message. Empty string when no prefix was set.

The core does NOT prepend `params.Prefix` into `Messages[0]`. Each transport decides how to render the prefix:

- **Fold the prefix into the message** (simplest): call `transport.JoinPrefixAndMessages(params.Prefix, params.Messages)` at the top of your `SendToLogger`. The helper returns `Messages` unchanged when `Prefix` is empty (fast path) or when `Messages[0]` isn't a string; otherwise it returns a fresh slice with `prefix + " "` prepended to `Messages[0]`. The output reads as one blob (`"[prefix] message body"`) which is what most renderer / wrapper transports want.
- **Render the prefix separately**: read `params.Prefix` directly and render it however suits your transport. A renderer can color the prefix differently from the message body; a structured transport can emit it as its own top-level field; a wrapper transport can forward it to the underlying logger's structured-field API (`zerolog.Event.Str("prefix", p.Prefix)`, etc.). Don't call `JoinPrefixAndMessages` in this path.

```go
func (t *Transport) SendToLogger(p loglayer.TransportParams) {
    if !t.ShouldProcess(p.LogLevel) {
        return
    }
    // Fold-into-message path:
    p.Messages = transport.JoinPrefixAndMessages(p.Prefix, p.Messages)
    // ... existing rendering ...
}
```

Use cases for reading `params.Prefix`:

- **Renderer transports** rendering the prefix in a different color than the message (e.g. dim `[auth]` + plain message body).
- **Structured transports** emitting the prefix as its own JSON field rather than embedding it in the message string.
- **Wrapper transports** forwarding the prefix to the underlying logger's structured-field mechanism.

## Level Filtering

`transport.BaseTransport.ShouldProcess(level)` returns false when:

- The transport is disabled (`SetEnabled(false)`), or
- The level is below the transport's `BaseConfig.Level`.

Always call it at the top of `SendToLogger`. The core LogLayer also filters by its own level state before reaching your transport, `ShouldProcess` is the second gate.

## Returning the Underlying Logger

If your transport wraps a third-party library, return that library from `GetLoggerInstance`. Callers can use it for backend-specific features the LogLayer API doesn't cover:

```go
func (t *Transport) GetLoggerInstance() any { return t.underlying }
```

For transports with no underlying library (anything you write from scratch), return `nil`.

## Concurrency

`SendToLogger` may be called from any goroutine. Make sure whatever you're writing to is safe, `os.Stdout`, `bytes.Buffer` (with a mutex), or a pre-locked `io.Writer`. The built-in transports rely on the writer being concurrency-safe; if yours isn't, wrap it.

## Don't Mutate TransportParams

When multiple transports are configured, they share the same `TransportParams`. Don't modify `params.Data`, `params.Messages`, or `params.Metadata` in place, copy first if you need to transform.

## Handling Errors

`SendToLogger` doesn't return an error. The dispatch path can't propagate transport failures back to the caller, so error handling is the transport's responsibility:

- **Synchronous renderer transports** (writing to `io.Writer`, terminal, file): if the write fails, there's nowhere to escalate. Print to `os.Stderr` and continue. Don't panic, don't `os.Exit`. A logging library that takes down the host process on a transient I/O hiccup is a bug.
- **Async / network transports** (HTTP, Datadog, anything with a worker goroutine and batching): expose an `OnError func(err error, ...)` field on your `Config` so the application can decide. Built-in transports follow this pattern, see [transports/http](/transports/http) and [transports/datadog](/transports/datadog) for the canonical shape.
- **Wrapper transports** (zerolog, zap, slog, etc.): the underlying library has its own error path (zap's `Sync` errors, zerolog's writer errors). Forward them or let the user reach the underlying logger via `GetLoggerInstance` and inspect there.

Transports that drop entries silently are valid: a logging library should never block, panic, or crash on its own write failure. But always make the failure observable somehow, even if it's just an `OnError` callback the user can hook into.

## Convention: package shape

If you publish a transport as a Go package, follow this shape:

```
yourpkg/
├── go.mod (if separate module)
├── yourpkg.go         // package yourpkg; exposes Config + Transport + New(Config) *Transport
├── errors.go          // (only if New can fail; sentinel errors)
├── yourpkg_test.go
└── README.md
```

The package name matches the directory name (`package yourpkg`). The main file owns:

- `Config` struct embedding `transport.BaseConfig`. Per-transport knobs live here.
- `Transport` struct embedding `transport.BaseTransport`.
- `func New(cfg Config) *Transport` as the typical entry point.

If `New` can fail with a runtime-loaded value (URL from env, API key from secrets manager, etc.), also expose `func Build(cfg Config) (*Transport, error)` and put your sentinel errors in `errors.go`. Name them `ErrXRequired` (`ErrURLRequired`, `ErrAPIKeyRequired`, ...) for consistency with `transports/http` and `transports/datadog`. Wrapper transports that take a pre-built `*zerolog.Logger` / `*zap.Logger` / etc. and have nothing to validate ship only `New`.

```go
// yourpkg.go
package yourpkg

import (
    "go.loglayer.dev/v2"
    "go.loglayer.dev/v2/transport"
)

type Config struct {
    transport.BaseConfig
    // ... your knobs
}

type Transport struct {
    transport.BaseTransport
    cfg Config
}

func New(cfg Config) *Transport { /* ... */ }

func (t *Transport) GetLoggerInstance() any { /* ... */ }
func (t *Transport) SendToLogger(p loglayer.TransportParams) { /* ... */ }
```

Match the pattern the built-ins use ([`transports/structured`](https://github.com/loglayer/loglayer-go/blob/main/transports/structured) for a renderer; [`transports/http`](https://github.com/loglayer/loglayer-go/blob/main/transports/http) for a network transport with `Build`).

## Testing

For testing a custom transport, see [Testing Transports](/transports/testing-transports). It covers the direct buffer assertion pattern and the `RunContract` helper that drives the same 14-test contract suite every built-in wrapper passes.
