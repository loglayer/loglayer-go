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

    "go.loglayer.dev"
    "go.loglayer.dev/transport"
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
    Messages []any   // already prefix-applied
    Data     Data    // assembled fields + error map; nil when both are absent. Use len(Data) > 0 to check.
    Metadata any     // raw value passed to WithMetadata, your transport decides serialization
    Err      error
    Fields   Fields
    Ctx      context.Context // per-call WithCtx value, or nil
}
```

`Data` is the convenience map combining fields + error. `Metadata` is `any`, you choose how to render it. `Err` and `Fields` are also exposed raw if you want to inspect them directly.

## Handling `any` Metadata

`params.Metadata` is whatever the caller passed to `WithMetadata`: a map, a struct, a pointer, a slice, a scalar, or `nil`. Your transport picks a **placement policy** for each shape. The common choices:

- Flatten a map's keys at the root of the log object.
- JSON-roundtrip a struct so its fields surface as root keys.
- Nest a non-map value under a single `metadata`-style key.
- Hand the raw value to an attribute-aware backend and let it serialize.

The `transport` package exposes helpers that encode each policy. Reach for them before writing your own type switch.

| Helper | What it does | Use when |
|--------|--------------|----------|
| `transport.MetadataAsRootMap(v) (map[string]any, bool)` | Returns the map directly if `v` is `loglayer.Metadata` or `map[string]any`; otherwise `nil, false`. No allocation, no roundtrip. | Deciding whether to flatten or nest. The wrapper transports (zap, zerolog, slog, logrus, charmlog, phuslu) call this first, then nest non-map values under `MetadataFieldName`. |
| `transport.MetadataAsMap(v) map[string]any` | Map fast path; non-map values are JSON-roundtripped into a map. Returns nil on roundtrip failure (channels, cycles, marshalers producing a non-object). | Renderers that flatten everything at the root, used by `structured` and (via `MergeFieldsAndMetadata`) by `console`. |
| `transport.MergeFieldsAndMetadata(p) map[string]any` | Combines `p.Data` and metadata into a single root-flat map. Map metadata merges at root; non-map is roundtripped via `MetadataAsMap` and dropped if the roundtrip fails. | Renderers that emit a single flat object. |
| `transport.MergeIntoMap(dst, data, metadata)` | Mutates `dst` in place. Map metadata merges at root; non-map metadata lands raw under the `metadata` key (no JSON roundtrip). | Encoders that have already seeded `dst` with their own protocol fields (level, time, msg, ddsource, ...) and want to layer user data on top. Used by HTTP `JSONArrayEncoder` and Datadog. |
| `transport.FieldEstimate(p) int` | Counts the eventual root-level fields. | Pre-sizing slices/maps in attribute-style backends (zap, charmlog, otellog). |

### Picking a policy

The right choice depends on what the backend can render natively:

- **Backends with no native struct support** (raw JSON, terminal lines): roundtrip via `MetadataAsMap` so struct fields surface as named root attributes. Slices and scalars don't roundtrip into a map, so decide explicitly: drop them (`structured`, `console`), fall back to a single key like `_metadata` (`pretty`), or nest them under a fixed key (`http`, `datadog`).
- **Backends with native attribute serializers** (zap, zerolog, slog, logrus, charmlog, phuslu, OpenTelemetry): use `MetadataAsRootMap` to flatten the map case, then hand non-map metadata directly to the backend's `Any` / `Interface` / `KeyValue` constructor under `MetadataFieldName`. Skip the roundtrip, the backend will serialize the value natively.

The built-in transports follow these two patterns so callers see consistent behavior across them. The contract those transports advertise on the [Metadata page](/logging-api/metadata) (map metadata flattens, struct metadata renders idiomatically per transport) is exactly what these helpers implement.

### Don't reinvent

Don't roll your own `metadataAsMap` unless your transport needs a placement policy these helpers don't already encode. The pretty transport's `_metadata` fallback is the only built-in example, and it's there because pretty is a human-readable renderer where dropping a slice silently is worse than showing it stringified.

## Reading `params.Ctx`

`params.Ctx` carries the `context.Context` the caller bound via [`WithCtx`](/logging-api/go-context). It's `nil` when no context was attached. Use it when your transport needs to forward the context to a downstream library (OpenTelemetry, slog handlers, anything context-aware) or extract values from it (trace IDs, deadlines, request-scoped data).

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
- **Self-contained renderers usually ignore it.** Pretty, structured, and console don't read context values themselves — that's a [plugin's](/plugins/creating-plugins) job. If you find yourself extracting trace IDs in a transport, prefer writing a plugin and pairing it with the transport: the plugin runs once per entry and feeds every transport, while transport-side extraction repeats per transport and bypasses the dispatch-time hook ordering.

If your transport extracts values from the context (rather than just forwarding it), test that path with a context that carries a sentinel value and assert the transport surfaced it.

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

## Testing your transport

Drive entries through your transport via a real `*loglayer.LogLayer` and assert on whatever your transport actually produced (a buffer, a captured request, a wrapped logger's calls). The pattern mirrors the built-in transport tests:

```go
import (
    "bytes"
    "testing"

    "go.loglayer.dev"
    "go.loglayer.dev/transport"
)

func TestMyTransport_Basic(t *testing.T) {
    buf := &bytes.Buffer{}
    tr := mytransport.New(mytransport.Config{
        BaseConfig: transport.BaseConfig{ID: "test"},
        Writer:     buf,
    })
    log := loglayer.New(loglayer.Config{
        Transport:        tr,
        DisableFatalExit: true,
    })

    log.WithFields(loglayer.Fields{"k": "v"}).Info("served")

    // Assert on whatever shape your transport produced.
    if !strings.Contains(buf.String(), `"k":"v"`) {
        t.Errorf("k=v missing from output: %q", buf.String())
    }
}
```

For wrapper transports (those that hand entries off to a third-party logger), assert on the wrapped logger's output rather than the transport's. The slog/zerolog/zap test files in `transports/` show this pattern.

Cover the level-filtering case, the `MetadataFieldName` non-map path, and `WithCtx` propagation when applicable. The existing wrapper-transport test files are good templates: same structure, same assertion shape.
